[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$gitCandidates = @(
    (Get-Command git -ErrorAction SilentlyContinue).Path,
    (Join-Path $env:USERPROFILE '.cache\codex-runtimes\codex-primary-runtime\dependencies\native\git\cmd\git.exe'),
    (Join-Path $env:USERPROFILE '.cache\codex-runtimes\codex-primary-runtime\dependencies\native\git\bin\git.exe')
) | Where-Object { $_ -and (Test-Path -LiteralPath $_) } | Select-Object -Unique
$gitCandidates = @($gitCandidates)

Write-Output ('RepositoryRoot=' + $repo)
if (!$gitCandidates) {
    Write-Output 'Git=missing'
} else {
    $git = $gitCandidates[0]
    Write-Output ('Git=' + $git)
    & $git -C $repo rev-parse --show-toplevel 2>$null | ForEach-Object { Write-Output ('GitRoot=' + $_) }
    & $git --version | ForEach-Object { Write-Output ('GitVersion=' + $_) }
    $remote = & $git -C $repo remote get-url origin 2>$null
    if ($remote) { Write-Output ('Origin=' + ($remote -replace 'https://[^/]+@','https://')) }
}

if (Get-Command gh -ErrorAction SilentlyContinue) { Write-Output 'GitHubCLI=present' } else { Write-Output 'GitHubCLI=not_required' }
Write-Output 'CredentialStatus=not_probed_by_default'
Write-Output 'ActionsRoute=GitHub Actions plus GitHub REST'
Write-Output 'Result=preflight_complete'
