$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
function Checked([string]$exe,[string[]]$arguments) { & $exe @arguments; if ($LASTEXITCODE -ne 0) { throw 'Installer build operation failed' } }
$compilerSetup = Join-Path $env:RUNNER_TEMP 'innosetup-6.7.4.exe'
Invoke-WebRequest https://github.com/jrsoftware/issrc/releases/download/is-6_7_4/innosetup-6.7.4.exe -OutFile $compilerSetup
if ((Get-FileHash $compilerSetup).Hash.ToLowerInvariant() -ne 'a549929319f9120716677c685535ca4671e4b937b5fdc87e2585f32d8eb30e8c') { throw 'Compiler checksum mismatch' }
$compilerDir = Join-Path $env:RUNNER_TEMP 'inno-6.7.4'
$process = Start-Process $compilerSetup -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$compilerDir+'"')) -Wait -PassThru
if ($process.ExitCode -ne 0) { throw 'Compiler provisioning failed' }
New-Item -ItemType Directory -Force dist | Out-Null
$goRoot = & go env GOROOT
$boltDir = & go list -m -f '{{.Dir}}' go.etcd.io/bbolt
$sysDir = & go list -m -f '{{.Dir}}' golang.org/x/sys
$licenses = @('Go 1.27.3', (Get-Content (Join-Path $goRoot 'LICENSE') -Raw), 'bbolt 1.4.3', (Get-Content (Join-Path $boltDir 'LICENSE') -Raw), 'golang.org/x/sys 0.49.0', (Get-Content (Join-Path $sysDir 'LICENSE') -Raw))
$licenses -join "`r`n`r`n" | Set-Content dist/THIRD-PARTY-LICENSES.txt
foreach ($version in @('0.1.0','0.1.1','0.1.2')) {
    Checked go @('build','-trimpath','-ldflags',('-s -w -X main.version='+$version+'-prototype'),'-o','dist/immichclone.exe','./cmd/immichclone')
    Checked "$compilerDir/ISCC.exe" @(('/DAppVersion='+$version),'packaging/immichclone.iss')
}
Get-ChildItem dist/*setup.exe | ForEach-Object { ((Get-FileHash $_.FullName).Hash.ToLowerInvariant()+'  '+$_.Name) } | Set-Content dist/SHA256SUMS.txt
