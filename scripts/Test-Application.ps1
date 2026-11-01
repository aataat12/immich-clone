$ErrorActionPreference = 'Stop'
$root = Join-Path $env:RUNNER_TEMP ('immich-app-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory $root | Out-Null
$binary = Join-Path $root 'immichclone.exe'
$state = Join-Path $root 'state'
$name = 'ImmichClone'
function Checked([string]$exe,[string[]]$arguments) { & $exe @arguments; if ($LASTEXITCODE -ne 0) { throw "$exe failed: $LASTEXITCODE" } }
function Assert([bool]$condition,[string]$message) { if (!$condition) { throw $message }; Write-Output "PASS: $message" }
function Login {
    $code = (Get-Content "$state/access-code.txt" -Raw).Trim()
    $script:session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $null = Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $script:session
}
try {
    Checked go @('build','-trimpath','-ldflags','-X main.version=0.1.0-prototype','-o',$binary,'./cmd/immichclone')
    Checked $binary @('-mode','init','-data',$state)
    Checked icacls @($root,'/grant','*S-1-5-19:(OI)(CI)RX')
    Checked icacls @($state,'/grant','*S-1-5-19:(OI)(CI)M')
    $credential = [PSCredential]::new('NT AUTHORITY\LocalService',(ConvertTo-SecureString 'unused' -AsPlainText -Force))
    New-Service -Name $name -BinaryPathName ('"'+$binary+'" -mode service -data "'+$state+'"') -Credential $credential | Out-Null
    $diagnosticRoot = Join-Path $env:ProgramData 'ImmichCloneDiagnostics'
    Checked icacls @($diagnosticRoot,'/grant','*S-1-5-19:(OI)(CI)M')
    Start-Service $name
    Assert ((Invoke-RestMethod http://127.0.0.1:2283/health).status -eq 'ok') 'actual application starts as LocalService'
    Login
    # Generate a fixture using the already-executed standard image encoders in tests.
    # This tiny known PNG fixture exercises actual upload/preview retrieval in service.
    $fixture = Join-Path $root 'synthetic.png'
    [IO.File]::WriteAllBytes($fixture,[Convert]::FromBase64String('iVBORw0KGgoAAAANSUhEUgAAAEAAAAAwCAIAAAAuKetIAAAAU0lEQVR4nO3PQQ3AIADAQEAbjhCLnYngcVnSU9DOfe74s6UDXjWgNaA1oDWgNaA1oDWgNaA1oDWgNaA1oDWgNaA1oDWgNaA1oDWgNaA1oDWgNaB9L9IB7tRAThgAAAAASUVORK5CYII='))
    $headers = @{'X-Immich-Request'='1'}
    $result = Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=synthetic.png' -Method Post -Headers $headers -InFile $fixture -WebSession $session
    Assert (!$result.duplicate -and $result.asset.id -eq (Get-FileHash $fixture).Hash.ToLowerInvariant()) 'upload returns original SHA256'
    $id = $result.asset.id
    $duplicate = Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=duplicate.png' -Method Post -Headers $headers -InFile $fixture -WebSession $session
    Assert $duplicate.duplicate 'duplicate upload resolves without overwriting original'
    $items = @()
    for ($attempt=0;$attempt -lt 15;$attempt++) {
        $items = @(Invoke-RestMethod http://127.0.0.1:2283/api/assets -WebSession $session)
        if ($items.Count -eq 1 -and $items[0].preview -ne 'queued') { break }
        Start-Sleep -Seconds 1
    }
    Assert ($items.Count -eq 1 -and $items[0].preview -eq 'ready') 'persistent preview job produces PNG under LocalService'
    $preview = Join-Path $root 'preview.png'
    Invoke-WebRequest "http://127.0.0.1:2283/api/assets/$id/preview" -WebSession $session -OutFile $preview
    Assert ((Get-Item $preview).Length -gt 0) 'generated preview can be retrieved'
    if ($env:IMMICH_BROWSER_MODULE) {
        $clip = Join-Path $root 'synthetic-video.mp4'
        ./scripts/New-MediaFixture.ps1 -Destination $clip
        $video = Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=synthetic-video.mp4' -Method Post -Headers $headers -InFile $clip -WebSession $session
        Assert ($video.asset.mime -eq 'video/mp4') 'actual H264/AAC MP4 upload recognized'
        $env:IMMICH_METADATA_FIXTURE = Join-Path $root 'oriented.jpg'
        Checked go @('test','./internal/library','-run','^TestMetadataFixtureEmitter$')
        $photo = Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=oriented.jpg&relative=SyntheticTrip%2Foriented.jpg' -Method Post -Headers $headers -InFile $env:IMMICH_METADATA_FIXTURE -WebSession $session
        Remove-Item Env:IMMICH_METADATA_FIXTURE
        Add-Type -AssemblyName System.Drawing
        $bitmap = [Drawing.Bitmap]::new(32,24)
        $graphics = [Drawing.Graphics]::FromImage($bitmap)
        $graphics.Clear([Drawing.Color]::Red)
        $codecOriginals = @{}
        foreach ($ext in @('Bmp','Tiff','Gif')) {
            $nameOnly = 'synthetic.'+$ext.ToLowerInvariant()
            $file = Join-Path $root $nameOnly
            $bitmap.Save($file,[Drawing.Imaging.ImageFormat]::$ext)
            $uploaded = Invoke-RestMethod ('http://127.0.0.1:2283/api/upload?name='+$nameOnly) -Method Post -Headers $headers -InFile $file -WebSession $session
            $codecOriginals[$uploaded.asset.id] = (Get-FileHash $file).Hash
        }
        $graphics.Dispose()
        $bitmap.Dispose()
        $env:IMMICH_TEST_STATE = $state
        Checked node @('tests/browser.cjs')
        Remove-Item Env:IMMICH_TEST_STATE
    }
    $onlineLog = Join-Path $root 'online-diagnostics.log'
    Invoke-WebRequest http://127.0.0.1:2283/api/diagnostics -WebSession $session -OutFile $onlineLog
    Assert ((Get-Content $onlineLog -Raw).Contains('original_write')) 'authenticated browser diagnostic export includes upload events'
    Restart-Service $name
    Login
    $download = Join-Path $root 'download.png'
    Invoke-WebRequest "http://127.0.0.1:2283/api/assets/$id/original" -WebSession $session -OutFile $download
    Assert ((Get-FileHash $download).Hash -eq (Get-FileHash $fixture).Hash) 'original download preserved across service restart'
    $backup = Join-Path $root 'backup.zip'
    Invoke-WebRequest http://127.0.0.1:2283/api/backup -WebSession $session -OutFile $backup
    Stop-Service $name
    $restored = Join-Path $root 'restored'
    Checked $binary @('-mode','restore','-data',$restored,'-archive',$backup)
    Checked $binary @('-mode','verify','-data',$restored)
    Assert ((Get-FileHash "$restored/originals/$id").Hash -eq (Get-FileHash $fixture).Hash) 'independent backup restoration verifies original bytes'
    if ($env:IMMICH_BROWSER_MODULE) {
        foreach ($entry in $codecOriginals.GetEnumerator()) {
            Assert ((Get-FileHash (Join-Path "$restored/originals" $entry.Key)).Hash -eq $entry.Value) 'BMP TIFF GIF original checksum preserved through independent restore'
        }
    }
    # Prove inability to write rather than relying on administrator test execution.
    Checked icacls @($state,'/deny','*S-1-5-19:(OI)(CI)W')
    $failed = $false
    try { Start-Service $name -ErrorAction Stop } catch { $failed = $true }
    Assert $failed 'service startup surfaces denied storage permissions'
    Checked icacls @($state,'/remove:d','*S-1-5-19')
    $support = Join-Path $root 'support.log'
    & "$env:SystemRoot/System32/WindowsPowerShell/v1.0/powershell.exe" -NoProfile -NonInteractive -ExecutionPolicy Bypass -File packaging/Export-Diagnostics.ps1 -OutputPath $support
    Assert ($LASTEXITCODE -eq 0 -and (Test-Path $support)) 'offline diagnostic export works while service is stopped'
    $diagnostic = Get-Content $support -Raw
    Assert ($diagnostic.Contains('permission_denied') -and $diagnostic.Contains('catalog_open')) 'startup permission failure is logged with useful category'
    Assert (!$diagnostic.Contains($root) -and !$diagnostic.Contains('synthetic.png') -and !$diagnostic.Contains((Get-Content "$state/access-code.txt" -Raw).Trim())) 'support log omits state paths media names and access code'
    Write-Output 'Application slice complete on Server 2022; Windows 10 remains untested.'
} finally {
    if (Get-Service $name -ErrorAction SilentlyContinue) { Stop-Service $name -Force -ErrorAction Continue; Checked sc.exe @('delete',$name) }
    Remove-Item $root -Recurse -Force -ErrorAction Continue
}
