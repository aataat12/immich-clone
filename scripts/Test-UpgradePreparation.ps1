param(
    [Parameter(Mandatory=$true)][string]$Binary,
    [string]$Compiler = ''
)
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
function Assert([bool]$Condition,[string]$Label) { if (!$Condition) { throw $Label }; Write-Output "PASS: $Label" }
function Native([string]$Exe,[string[]]$Arguments) { & $Exe @Arguments | Out-Null; if ($LASTEXITCODE -ne 0) { throw 'Preparation probe native operation failed' } }
function Wait-Health {
    for ($i=0;$i -lt 30;$i++) {
        try { if ((Invoke-RestMethod http://127.0.0.1:2283/health).status -eq 'ok') { return } } catch { }
        Start-Sleep -Milliseconds 500
    }
    throw 'Preparation probe service did not become healthy'
}
function Run-Setup([string]$Application,[string]$State) {
    $process = Start-Process $script:setup -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$Application+'"'),('/STATE="'+$State+'"')) -Wait -PassThru
    return $process.ExitCode
}
function Write-RetainedHelper([string]$Application) {
    @'
param([string]$Action,[string]$AppDir,[string]$DataDir)
[IO.File]::WriteAllText((Join-Path $PSScriptRoot 'retained-preparer-ran.txt'),'unexpected retained preparer execution')
exit 0
'@ | Set-Content -LiteralPath (Join-Path $Application 'Manage-Service.ps1')
}
$root = Join-Path $env:RUNNER_TEMP ('upgrade preparation '+[guid]::NewGuid().ToString('N')+' é')
$repo = Split-Path $PSScriptRoot -Parent
$currentApp = ''
$junction = ''
$serviceOwned = $false
try {
    Assert (!(Get-Service ImmichClone -ErrorAction SilentlyContinue)) 'upgrade preparation probe refuses an existing service'
    Assert (!(Test-Path 'HKLM:\SOFTWARE\ImmichClone')) 'upgrade preparation probe refuses existing application registration'
    Assert (!(Test-Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\{5DF7680B-6F42-4BAF-AD97-5A3095728DFA}_is1')) 'upgrade preparation probe refuses existing installer registration'
    New-Item -ItemType Directory -Path $root | Out-Null
    Native icacls @($root,'/grant','*S-1-5-19:(OI)(CI)RX')
    if (!$Compiler) {
        $compilerSetup = Join-Path $root 'innosetup-6.7.4.exe'
        Invoke-WebRequest 'https://github.com/jrsoftware/issrc/releases/download/is-6_7_4/innosetup-6.7.4.exe' -OutFile $compilerSetup
        Assert ((Get-FileHash $compilerSetup).Hash.ToLowerInvariant() -eq 'a549929319f9120716677c685535ca4671e4b937b5fdc87e2585f32d8eb30e8c') 'pinned Inno 6.7.4 compiler archive hash verified'
        $compilerDir = Join-Path $root 'inno-6.7.4'
        $process = Start-Process $compilerSetup -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$compilerDir+'"')) -Wait -PassThru
        Assert ($process.ExitCode -eq 0) 'pinned Inno compiler provisioned for disposable probe'
        $Compiler = Join-Path $compilerDir 'ISCC.exe'
    }
    # Compile the actual production .iss, not a duplicate of its Pascal logic.
    # Minimal fixture providers are never delivered or uploaded as release artifacts.
    $source = Join-Path $root 'source'
    foreach ($directory in @('packaging','docs','dist','dist/media','dist/ai')) { New-Item -ItemType Directory -Force (Join-Path $source $directory) | Out-Null }
    foreach ($file in @('immichclone.iss','Manage-Service.ps1','Storage-Security.ps1','Export-Diagnostics.ps1')) { Copy-Item -LiteralPath (Join-Path $repo ('packaging/'+$file)) -Destination (Join-Path $source ('packaging/'+$file)) }
    Copy-Item -LiteralPath (Join-Path $repo 'docs/prototype-notes.txt') -Destination (Join-Path $source 'docs/prototype-notes.txt')
    Copy-Item -LiteralPath $Binary -Destination (Join-Path $source 'dist/immichclone.exe')
    foreach ($file in @('dist/THIRD-PARTY-LICENSES.txt','dist/media/preparation-probe.txt','dist/ai/preparation-probe.txt')) { [IO.File]::WriteAllText((Join-Path $source $file),'Disposable upgrade-preparation fixture only; not a distributable package.') }
    Native $Compiler @('/DAppVersion=0.0.0',(Join-Path $source 'packaging/immichclone.iss'))
    $script:setup = Join-Path $source 'dist/ImmichClone-0.0.0-prototype-x64-setup.exe'
    Assert (Test-Path -LiteralPath $script:setup) 'production Inno script compiled with minimal disposable payload'

    # A static junction must abort before touching retained app files. The old
    # helper would write a harmless marker if it were mistakenly invoked.
    $negativeApp = Join-Path $root 'rejected application'
    $outside = Join-Path $root 'outside'
    New-Item -ItemType Directory -Path $negativeApp,$outside | Out-Null
    Write-RetainedHelper $negativeApp
    $oldBinary = Join-Path $negativeApp 'immichclone.exe'
    [IO.File]::WriteAllText($oldBinary,'retained binary sentinel')
    $oldBinaryHash = (Get-FileHash $oldBinary).Hash
    $oldHelperHash = (Get-FileHash (Join-Path $negativeApp 'Manage-Service.ps1')).Hash
    $sentinel = Join-Path $outside 'sentinel.txt'; [IO.File]::WriteAllText($sentinel,'outside sentinel')
    $outsideAcl = (Get-Acl -LiteralPath $outside).Sddl
    $junction = Join-Path $root 'redirected state'
    New-Item -ItemType Junction -Path $junction -Target $outside | Out-Null
    Assert ((Run-Setup $negativeApp $junction) -ne 0) 'actual Setup rejects junction state during bundled preparation'
    Assert (!(Test-Path (Join-Path $negativeApp 'retained-preparer-ran.txt'))) 'rejected preparation never executes retained old helper'
    Assert ((Get-FileHash $oldBinary).Hash -eq $oldBinaryHash -and (Get-FileHash (Join-Path $negativeApp 'Manage-Service.ps1')).Hash -eq $oldHelperHash -and !(Test-Path (Join-Path $negativeApp 'Storage-Security.ps1'))) 'preparation failure publishes no replacement application files'
    Assert ((Get-Acl -LiteralPath $outside).Sddl -eq $outsideAcl -and [IO.File]::ReadAllText($sentinel) -eq 'outside sentinel' -and @(Get-ChildItem -LiteralPath $outside -Force).Count -eq 1) 'rejected Setup leaves junction target ACL and bytes unchanged'
    Assert (!(Get-Service ImmichClone -ErrorAction SilentlyContinue) -and !(Test-Path 'HKLM:\SOFTWARE\ImmichClone')) 'rejected preparation registers no service or state'
    [IO.Directory]::Delete($junction); $junction = ''

    # Create an actual service-owned existing library without using the retained
    # helper. It deliberately represents old files, not an older runtime version.
    $currentApp = Join-Path $root 'existing application'
    $state = Join-Path $root 'existing state'
    New-Item -ItemType Directory -Path $currentApp | Out-Null
    Copy-Item -LiteralPath $Binary -Destination (Join-Path $currentApp 'immichclone.exe')
    Write-RetainedHelper $currentApp
    Native (Join-Path $currentApp 'immichclone.exe') @('-mode','init','-data',$state)
    Native icacls @($state,'/grant','*S-1-5-19:(OI)(CI)M')
    $credential = New-Object PSCredential('NT AUTHORITY\LocalService',(ConvertTo-SecureString 'unused' -AsPlainText -Force))
    New-Service -Name ImmichClone -BinaryPathName ('"'+(Join-Path $currentApp 'immichclone.exe')+'" -mode service -data "'+$state+'"') -Credential $credential | Out-Null
    $serviceOwned = $true
    Start-Service ImmichClone
    Wait-Health
    $code = ([IO.File]::ReadAllText((Join-Path $state 'access-code.txt'))).Trim()
    $codeHash = (Get-FileHash (Join-Path $state 'access-code.txt')).Hash
    $session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $null = Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $session
    $original = Join-Path $root 'synthetic-original.bin'; [IO.File]::WriteAllText($original,'upgrade preparation synthetic original')
    $asset = (Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=fixture.bin' -Method Post -Headers @{'X-Immich-Request'='1'} -InFile $original -WebSession $session).asset
    $originalHash = (Get-FileHash $original).Hash
    $priorBinaryHash = (Get-FileHash (Join-Path $currentApp 'immichclone.exe')).Hash
    Assert ((Run-Setup $currentApp $state) -eq 0) 'actual Setup succeeds using bundled preparation against existing LocalService state'
    Wait-Health
    Assert (!(Test-Path (Join-Path $currentApp 'retained-preparer-ran.txt'))) 'successful preparation never executes retained old helper'
    Assert ((Get-FileHash (Join-Path $currentApp 'rollback/immichclone.exe')).Hash -eq $priorBinaryHash -and (Get-FileHash (Join-Path $currentApp 'rollback/access-code.txt')).Hash -eq $codeHash -and (Test-Path (Join-Path $currentApp 'rollback/catalog.db'))) 'bundled preparation snapshots real prior binary catalog and access code'
    Assert ((Get-FileHash (Join-Path $currentApp 'Manage-Service.ps1')).Hash -eq (Get-FileHash (Join-Path $source 'packaging/Manage-Service.ps1')).Hash -and (Get-FileHash (Join-Path $currentApp 'Storage-Security.ps1')).Hash -eq (Get-FileHash (Join-Path $source 'packaging/Storage-Security.ps1')).Hash) 'Setup installs exact candidate management and sibling helper bytes'
    Assert ((Get-FileHash (Join-Path $state ('originals/'+$asset.id))).Hash -eq $originalHash -and (Get-FileHash (Join-Path $state 'access-code.txt')).Hash -eq $codeHash) 'bundled preparation preserves managed original and access code'
    Assert ((Get-CimInstance Win32_Service -Filter "Name='ImmichClone'").StartName -eq 'NT AUTHORITY\LocalService') 'prepared service remains LocalService'
    Assert ((Get-ItemProperty 'HKLM:\SOFTWARE\ImmichClone').State -eq $state) 'Setup retains real application data location'
    $process = Start-Process (Join-Path $currentApp 'unins000.exe') -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru
    Assert ($process.ExitCode -eq 0) 'disposable preparation installer uninstalls cleanly'
    Assert ((Get-FileHash (Join-Path $state ('originals/'+$asset.id))).Hash -eq $originalHash) 'probe uninstall retains managed original bytes'
} finally {
    # Never remove an unrelated service/registration if this fixture did not own it.
    if ($serviceOwned) {
        $service = Get-CimInstance Win32_Service -Filter "Name='ImmichClone'" -ErrorAction SilentlyContinue
        if ($service -and $service.PathName.StartsWith(('"'+(Join-Path $currentApp 'immichclone.exe')+'"'),[StringComparison]::OrdinalIgnoreCase)) { Stop-Service ImmichClone -Force -ErrorAction SilentlyContinue; & sc.exe delete ImmichClone | Out-Null }
    }
    if ($currentApp -and (Test-Path (Join-Path $currentApp 'unins000.exe'))) { $null = Start-Process (Join-Path $currentApp 'unins000.exe') -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru }
    if ($junction -and (Test-Path -LiteralPath $junction)) { [IO.Directory]::Delete($junction) }
    if (Test-Path -LiteralPath $root) { Remove-Item -LiteralPath $root -Recurse -Force }
}
