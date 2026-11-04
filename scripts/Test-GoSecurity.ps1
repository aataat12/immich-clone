$ErrorActionPreference = 'Stop'
# Tool-only dependency: not part of the installed application or its go.mod.
$env:GOBIN = Join-Path $env:RUNNER_TEMP 'security-tools'
$env:GOTOOLCHAIN = 'local'
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
$env:CGO_ENABLED = '0'
go install golang.org/x/vuln/cmd/govulncheck@v1.9.0
if ($LASTEXITCODE -ne 0) { throw 'Pinned security scanner provisioning failed' }
$scanner = Join-Path $env:GOBIN 'govulncheck.exe'
& $scanner -version
if ($LASTEXITCODE -ne 0) { throw 'Scanner identity check failed' }
& $scanner ./cmd/immichclone ./cmd/installerfiles
if ($LASTEXITCODE -ne 0) { throw 'Dependency vulnerability check failed; inspect findings before release' }
