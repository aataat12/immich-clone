param([string[]]$Versions=@('0.1.43','0.1.44'),[string]$NotesPath='docs/release-notes-0.1.43.md')
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
# All three accepted features must be enabled before producing a release candidate.
$automaticNamesSource=Get-Content internal/library/automatic_names.go -Raw
if($automaticNamesSource -notmatch '(?m)^const automaticNamesValidated = true\s*$'){throw '0.1.43 packaging is gated on independent automatic naming acceptance and explicit production enablement'}
function Checked([string]$exe,[string[]]$arguments) { & $exe @arguments; if ($LASTEXITCODE -ne 0) { throw 'Installer build operation failed' } }
$compilerSetup = Join-Path $env:RUNNER_TEMP 'innosetup-6.7.4.exe'
Invoke-WebRequest https://github.com/jrsoftware/issrc/releases/download/is-6_7_4/innosetup-6.7.4.exe -OutFile $compilerSetup
if ((Get-FileHash $compilerSetup).Hash.ToLowerInvariant() -ne 'a549929319f9120716677c685535ca4671e4b937b5fdc87e2585f32d8eb30e8c') { throw 'Compiler checksum mismatch' }
$compilerDir = Join-Path $env:RUNNER_TEMP 'inno-6.7.4'
$process = Start-Process $compilerSetup -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$compilerDir+'"')) -Wait -PassThru -WindowStyle Hidden
if ($process.ExitCode -ne 0) { throw 'Compiler provisioning failed' }
New-Item -ItemType Directory -Force dist | Out-Null
Copy-Item -LiteralPath $NotesPath -Destination dist/prototype-notes.txt
./scripts/Prepare-MediaProvider.ps1 -Destination 'dist/media/ffmpeg-8.1.3-7eb77d88e2e9'
./scripts/Prepare-AIProvider.ps1 -Destination 'dist/ai/opencv-4.12.0-nr01-zlib-1.3.2-15d1c2ca7ef3'
$goRoot = & go env GOROOT
$boltDir = & go list -m -f '{{.Dir}}' go.etcd.io/bbolt
$sysDir = & go list -m -f '{{.Dir}}' golang.org/x/sys
$imageDir = & go list -m -f '{{.Dir}}' golang.org/x/image
$licenses = @('Go 1.27.3', (Get-Content (Join-Path $goRoot 'LICENSE') -Raw), 'bbolt 1.4.3', (Get-Content (Join-Path $boltDir 'LICENSE') -Raw), 'golang.org/x/sys 0.49.0', (Get-Content (Join-Path $sysDir 'LICENSE') -Raw))
$licenses += @('golang.org/x/image 0.47.0', (Get-Content (Join-Path $imageDir 'LICENSE') -Raw))
$licenses -join "`r`n`r`n" | Set-Content dist/THIRD-PARTY-LICENSES.txt
Checked go @('build','-trimpath','-ldflags','-s -w','-o','dist/installer-files.exe','./cmd/installerfiles')
foreach ($version in $Versions) {
    Checked go @('build','-trimpath','-ldflags',('-s -w -X main.version='+$version+'-prototype'),'-o','dist/immichclone.exe','./cmd/immichclone')
    Checked "$compilerDir/ISCC.exe" @(('/DAppVersion='+$version),'packaging/immichclone.iss')
}
Get-Item ("dist/ImmichClone-"+$Versions[0]+"-prototype-x64-setup.exe") | ForEach-Object { ((Get-FileHash $_.FullName).Hash.ToLowerInvariant()+'  '+$_.Name) } | Set-Content dist/SHA256SUMS.txt
