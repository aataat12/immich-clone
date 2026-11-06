$ErrorActionPreference='Stop'
$ProgressPreference='SilentlyContinue'
if((Get-Content internal/library/automatic_names.go -Raw) -notmatch '(?m)^const automaticNamesValidated = true\s*$'){throw 'Actual native automatic lifecycle requires production enablement after independent quality acceptance'}
$root=Join-Path $env:RUNNER_TEMP ('actual-automatic-'+[guid]::NewGuid().ToString('N'))
$previous=$env:IMMICH_ACTUAL_AUTOMATIC_ROOT
try {
 New-Item -ItemType Directory -Path $root|Out-Null
 ./scripts/Prepare-AIProvider.ps1 -Destination (Join-Path $root 'ai')
 $fixture=Join-Path $root 'astronaut.png'
 Invoke-WebRequest 'https://raw.githubusercontent.com/scikit-image/scikit-image/e8a42ba85aaf5fd9322ef9ca51bc21063b22fcae/skimage/data/astronaut.png' -OutFile $fixture
 if((Get-FileHash -LiteralPath $fixture).Hash.ToLowerInvariant() -ne '88431cd9653ccd539741b555fb0a46b61558b301d4110412b5bc28b5e3ea6cb5'){throw 'Approved public functional fixture checksum mismatch'}
 $env:IMMICH_ACTUAL_AUTOMATIC_ROOT=$root
 go test -mod=readonly -v -count=1 -timeout=5m ./internal/library -run '^TestActualAutomaticNamesPublicationLifecycle$'
 if($LASTEXITCODE -ne 0){throw 'Actual native automatic naming lifecycle failed'}
} finally {
 $env:IMMICH_ACTUAL_AUTOMATIC_ROOT=$previous
 if(Test-Path -LiteralPath $root){
  $target=[IO.Path]::GetFullPath($root)
  $parent=[IO.Path]::GetFullPath($env:RUNNER_TEMP).TrimEnd('\')
  if([IO.Path]::GetDirectoryName($target) -cne $parent -or [IO.Path]::GetFileName($target) -notmatch '^actual-automatic-[0-9a-f]{32}$'){throw 'Actual native lifecycle cleanup target refused'}
  Remove-Item -LiteralPath $target -Recurse -Force
 }
}
