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
    ./scripts/Prepare-MediaProvider.ps1 -Destination (Join-Path $root 'media/ffmpeg-8.1.3-7eb77d88e2e9')
    ./scripts/Prepare-AIProvider.ps1 -Destination (Join-Path $root 'ai/opencv-4.12.0-a83dcef02ebe')
    Checked go @('build','-trimpath','-ldflags','-X main.version=0.1.25-prototype','-o',$binary,'./cmd/immichclone')
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
        $items = Invoke-RestMethod http://127.0.0.1:2283/api/assets -WebSession $session
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
        $mediaAssets = @{}
        foreach ($fixtureName in @('synthetic.heic','grid-large.heic','hevc.mp4','rotated.mov')) {
            $file = Join-Path $root $fixtureName
            [IO.File]::WriteAllBytes($file,[Convert]::FromBase64String((Get-Content (Join-Path 'experiments/media-worker/fixtures' ($fixtureName+'.b64')) -Raw)))
            $uploaded = Invoke-RestMethod ('http://127.0.0.1:2283/api/upload?name='+$fixtureName) -Method Post -Headers $headers -InFile $file -WebSession $session
            $mediaAssets[$fixtureName] = $uploaded.asset.id
            $codecOriginals[$uploaded.asset.id] = (Get-FileHash $file).Hash
        }
        $env:IMMICH_TEST_STATE = $state
        Checked node @('tests/browser.cjs')
        Checked node @('tests/people.cjs')
        Checked node @('tests/import.cjs')
        Remove-Item Env:IMMICH_TEST_STATE
    }
    if ($env:IMMICH_BROWSER_MODULE) {
        Stop-Service $name
        $providerExe = Join-Path $root 'media/ffmpeg-8.1.3-7eb77d88e2e9/ffmpeg.exe'
        Move-Item $providerExe ($providerExe+'.held')
        Start-Service $name
        Login
        $gridId = $mediaAssets['grid-large.heic']
        $null = Invoke-RestMethod "http://127.0.0.1:2283/api/assets/$gridId/retry" -Method Post -Headers $headers -WebSession $session
        $null = Invoke-RestMethod "http://127.0.0.1:2283/api/assets/$id/retry" -Method Post -Headers $headers -WebSession $session
        for ($i=0;$i -lt 15;$i++) {
            $items = Invoke-RestMethod http://127.0.0.1:2283/api/assets -WebSession $session
            $grid = $items | Where-Object id -eq $gridId
            $native = $items | Where-Object id -eq $id
            if ($grid.preview -eq 'failed' -and $native.preview -eq 'ready') { break }
            Start-Sleep -Seconds 1
        }
        Write-Output ('Synthetic optional-provider states: '+(@{grid=$grid.preview;native=$native.preview;count=$items.Count}|ConvertTo-Json -Compress))
        Assert ($grid.preview -eq 'failed' -and $native.preview -eq 'ready' -and (Invoke-RestMethod http://127.0.0.1:2283/health).status -eq 'ok') 'missing optional provider leaves native previews storage and HTTP operational'
        Stop-Service $name
        Move-Item ($providerExe+'.held') $providerExe
        Start-Service $name
        Login
        $null = Invoke-RestMethod "http://127.0.0.1:2283/api/assets/$gridId/retry" -Method Post -Headers $headers -WebSession $session
        for ($i=0;$i -lt 15;$i++) {
            $grid = (Invoke-RestMethod http://127.0.0.1:2283/api/assets -WebSession $session) | Where-Object id -eq $gridId
            if ($grid.preview -eq 'ready') { break }
            Start-Sleep -Seconds 1
        }
        Assert ($grid.preview -eq 'ready' -and $grid.width -eq 4032 -and $grid.height -eq 3024) 'optional provider recovery retries HEIC and preserves source dimensions'
    }
    if ($env:IMMICH_BROWSER_MODULE) {
        Stop-Service $name
        $aiModel=Join-Path $root 'ai/opencv-4.12.0-a83dcef02ebe/sface.onnx'
        Move-Item $aiModel ($aiModel+'.held')
        Start-Service $name;Login
        $org=Invoke-RestMethod http://127.0.0.1:2283/api/organization -WebSession $session
        Assert (!$org.available -and (Invoke-RestMethod http://127.0.0.1:2283/health).status -eq 'ok') 'missing AI model leaves library and saved organization available'
        # Bulk analysis has already processed the original fixture. Use a fresh
        # valid PNG so this failure probe does not accidentally request reanalysis.
        $aiFixture=Join-Path $root 'synthetic-ai-recovery.png'
        $aiBitmap=[Drawing.Bitmap]::new(64,48)
        $aiGraphics=[Drawing.Graphics]::FromImage($aiBitmap)
        try { $aiGraphics.Clear([Drawing.Color]::Blue);$aiBitmap.Save($aiFixture,[Drawing.Imaging.ImageFormat]::Png) }
        finally { $aiGraphics.Dispose();$aiBitmap.Dispose() }
        $aiUpload=Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=synthetic-ai-recovery.png' -Method Post -Headers $headers -InFile $aiFixture -WebSession $session
        Assert (!$aiUpload.duplicate) 'missing-model probe uses a new unprocessed original'
        $aiId=$aiUpload.asset.id
        $null=Invoke-RestMethod "http://127.0.0.1:2283/api/assets/$aiId/analyze" -Method Post -Headers $headers -WebSession $session
        for($i=0;$i -lt 15;$i++) { $org=Invoke-RestMethod http://127.0.0.1:2283/api/organization -WebSession $session; $ai=$org.assets | Where-Object { $_.asset.id -eq $aiId }; if($ai.analysis.status -eq 'failed'){break};Start-Sleep -Seconds 1 }
        Assert ($ai.analysis.status -eq 'failed') 'AI failure is visible and original access is retained'
        $aiOriginal=Join-Path $root 'synthetic-ai-recovery-returned.png'
        Invoke-WebRequest "http://127.0.0.1:2283/api/assets/$aiId/original" -WebSession $session -OutFile $aiOriginal
        Assert ((Get-FileHash $aiOriginal).Hash -eq (Get-FileHash $aiFixture).Hash) 'missing-model failure preserves original download bytes'
        Stop-Service $name;Move-Item ($aiModel+'.held') $aiModel;Start-Service $name;Login
        $null=Invoke-RestMethod "http://127.0.0.1:2283/api/assets/$aiId/analyze" -Method Post -Headers $headers -WebSession $session
        for($i=0;$i -lt 30;$i++) { $org=Invoke-RestMethod http://127.0.0.1:2283/api/organization -WebSession $session; $ai=$org.assets | Where-Object { $_.asset.id -eq $aiId }; if($ai.analysis.status -ne 'queued'){break};Start-Sleep -Seconds 1 }
        Assert ($ai.analysis.status -eq 'ready') 'AI provider recovery retries the failed fixture successfully'
        Assert (@($org.faces | Where-Object name -eq 'Test Person').Count -eq 1) 'AI provider recovery preserves manual names'
    }
    if ($env:IMMICH_BROWSER_MODULE) { $env:IMMICH_TEST_STATE=$state;Checked node @('tests/gallery.cjs');Remove-Item Env:IMMICH_TEST_STATE }
    if ($env:IMMICH_BROWSER_MODULE) { $env:IMMICH_TEST_STATE=$state;Checked node @('tests/backup.cjs');Remove-Item Env:IMMICH_TEST_STATE }
    $onlineLog = Join-Path $root 'online-diagnostics.log'
    Invoke-WebRequest http://127.0.0.1:2283/api/diagnostics -WebSession $session -OutFile $onlineLog
    Assert ((Get-Content $onlineLog -Raw).Contains('original_write')) 'authenticated browser diagnostic export includes upload events'
    Assert ((Get-Content $onlineLog -Raw).Contains('media_provider_unavailable') -and (Get-Content $onlineLog -Raw).Contains('media_probe') -and (Get-Content $onlineLog -Raw).Contains('media_decode')) 'private diagnostics distinguish provider absence and media stages'
    Restart-Service $name
    Login
    $download = Join-Path $root 'download.png'
    Invoke-WebRequest "http://127.0.0.1:2283/api/assets/$id/original" -WebSession $session -OutFile $download
    Assert ((Get-FileHash $download).Hash -eq (Get-FileHash $fixture).Hash) 'original download preserved across service restart'
    if ($env:IMMICH_BROWSER_MODULE) {
        $imports=Invoke-RestMethod http://127.0.0.1:2283/api/imports -WebSession $session
        Assert (@($imports | Where-Object { $_.status -eq 'completed' -and $_.accepted_files -eq 4 }).Count -eq 1 -and @($imports | Where-Object { $_.status -eq 'cancelled' -and $_.accepted_files -eq 1 }).Count -eq 1) 'service restarts retain completed and cancelled import receipts'
    }
    if ($env:IMMICH_BROWSER_MODULE) {
        $backupJob=Invoke-RestMethod http://127.0.0.1:2283/api/backup-job -WebSession $session
        $managedArchive=Join-Path $env:IMMICH_BROWSER_OUTPUT 'managed-backup.zip'
        Assert ($backupJob.status -eq 'ready' -and $backupJob.sha256 -eq (Get-FileHash $managedArchive).Hash.ToLowerInvariant()) 'completed verified backup remains downloadable after service restart'
        $retainedPrefix=Join-Path $root 'retained-backup-prefix.bin'
        $rangeSession=[Microsoft.PowerShell.Commands.WebRequestSession]::new();$rangeSession.Cookies=$session.Cookies
        $rangeResult=Invoke-WebRequest ("http://127.0.0.1:2283/api/backup-job/download?id="+$backupJob.id) -Headers @{Range='bytes=0-15'} -WebSession $rangeSession -OutFile $retainedPrefix -PassThru
        Assert ($rangeResult.StatusCode -eq 206 -and (Get-Item $retainedPrefix).Length -eq 16) 'retained backup supports real range download after restart'
        Assert (!$session.Headers.ContainsKey('Range')) 'range probe cannot contaminate subsequent full downloads'
        $env:IMMICH_TEST_STATE=$state;Checked node @('tests/backup.cjs','--discard');Remove-Item Env:IMMICH_TEST_STATE
    }
    $backup = Join-Path $root 'backup.zip'
    $backupResponse=Invoke-WebRequest http://127.0.0.1:2283/api/backup -WebSession $session -OutFile $backup -PassThru
    Assert ($backupResponse.StatusCode -eq 200 -and (Get-Item $backup).Length -gt 16) 'legacy backup download is a complete response before restore'
    Stop-Service $name
    $restored = Join-Path $root 'restored'
    Checked $binary @('-mode','restore','-data',$restored,'-archive',$backup)
    Checked $binary @('-mode','verify','-data',$restored)
    if ($env:IMMICH_BROWSER_MODULE) { $managedRestored=Join-Path $root 'managed-restored';Checked $binary @('-mode','restore','-data',$managedRestored,'-archive',$managedArchive);Checked $binary @('-mode','verify','-data',$managedRestored) }
    Assert ((Get-FileHash "$restored/originals/$id").Hash -eq (Get-FileHash $fixture).Hash) 'independent backup restoration verifies original bytes'
    if ($env:IMMICH_BROWSER_MODULE) {
        foreach ($entry in $codecOriginals.GetEnumerator()) {
            Assert ((Get-FileHash (Join-Path "$restored/originals" $entry.Key)).Hash -eq $entry.Value) 'image and video original checksum preserved through independent restore'
        }
    }
    if ($env:IMMICH_BROWSER_MODULE) {
        Move-Item $state ($state+'.offline')
        try {
            foreach ($restoreRoot in @($restored,$managedRestored)) {
                $env:IMMICH_ORGANIZATION_RESTORE=$restoreRoot
                Checked go @('test','./internal/library','-run','^Test(Organization|Import)RestoredCatalogCheck$','-count=1')
            }

            Assert $true 'independent restored names tags and face crops work without old state or AI provider'
        } finally { Remove-Item Env:IMMICH_ORGANIZATION_RESTORE -ErrorAction SilentlyContinue;Move-Item ($state+'.offline') $state }
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
