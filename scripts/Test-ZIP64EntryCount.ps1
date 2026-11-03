$ErrorActionPreference='Stop'
$priorEnabled=$env:IMMICH_ZIP64_COUNT
$priorTemp=$env:IMMICH_ZIP64_TEMP
try {
    $env:IMMICH_ZIP64_COUNT='1'
    $env:IMMICH_ZIP64_TEMP=$env:RUNNER_TEMP
    go test -mod=readonly -v -count=1 -timeout=17m ./internal/library -run '^TestActualZIP64EntryCountBackupRestore$'
    if ($LASTEXITCODE -ne 0) { throw 'Actual ZIP64 entry-count backup/restore acceptance failed' }
} finally {
    $env:IMMICH_ZIP64_COUNT=$priorEnabled
    $env:IMMICH_ZIP64_TEMP=$priorTemp
}
