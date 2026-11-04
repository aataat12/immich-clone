param([string]$TemporaryParent=$env:RUNNER_TEMP)
$ErrorActionPreference='Stop'
if(!$TemporaryParent -or !(Test-Path -LiteralPath $TemporaryParent -PathType Container)){throw 'Choose an existing disposable test parent with at least 12 GiB free; no source media is used'}
$oldFlag=$env:IMMICH_IMPORT_BOUNDARY
$oldParent=$env:IMMICH_IMPORT_BOUNDARY_TEMP
try {
 $env:IMMICH_IMPORT_BOUNDARY='1'
 $env:IMMICH_IMPORT_BOUNDARY_TEMP=(Resolve-Path -LiteralPath $TemporaryParent).Path
 go test -mod=readonly -v -count=1 -timeout=10m ./internal/web -run '^TestImportExact4GiBHTTPBoundary$'
 if($LASTEXITCODE -ne 0){throw 'Actual HTTP import boundary acceptance failed'}
} finally {
 if($null -eq $oldFlag){Remove-Item Env:IMMICH_IMPORT_BOUNDARY -ErrorAction SilentlyContinue}else{$env:IMMICH_IMPORT_BOUNDARY=$oldFlag}
 if($null -eq $oldParent){Remove-Item Env:IMMICH_IMPORT_BOUNDARY_TEMP -ErrorAction SilentlyContinue}else{$env:IMMICH_IMPORT_BOUNDARY_TEMP=$oldParent}
}
