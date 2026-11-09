[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$Manifest,
    [switch]$Publish
)

$ErrorActionPreference = 'Stop'
$repo = 'aataat12/immich-clone'
$api = "https://api.github.com/repos/$repo"
$tag = 'v0.1.44-prototype'
$accepted = Get-Content -LiteralPath $Manifest -Raw | ConvertFrom-Json
if ($accepted.source -cnotmatch '^[a-f0-9]{40}$' -or
    $accepted.archive_sha256 -cnotmatch '^[a-f0-9]{64}$' -or
    [long]$accepted.run -le 0 -or [long]$accepted.artifact -le 0) {
    throw 'The acceptance manifest must pin the exact commit, run, artifact and archive digest'
}
if (!$env:GH_TOKEN) { throw 'Use an authenticated GitHub release session with GH_TOKEN' }
$headers = @{
    Accept = 'application/vnd.github+json'
    Authorization = "Bearer $env:GH_TOKEN"
    'X-GitHub-Api-Version' = '2022-11-28'
}

function Assert-ReleaseAssets($assets) {
    if (@($assets).Count -ne $files.Count) { throw 'Unexpected release asset count' }
    foreach ($file in $files) {
        $asset = @($assets | Where-Object name -CEQ $file.Name)
        if ($asset.Count -ne 1 -or $asset[0].state -ne 'uploaded' -or
            [long]$asset[0].size -ne $file.Length -or $asset[0].digest -cne "sha256:$($file.Hash)") {
            throw "GitHub asset does not match the accepted bytes: $($file.Name)"
        }
    }
}

$run = Invoke-RestMethod "$api/actions/runs/$($accepted.run)" -Headers $headers
if ($run.head_sha -cne $accepted.source -or $run.head_branch -cne 'work/release-0.1.44-foundation' -or
    $run.path -cne '.github/workflows/release44-acceptance.yml' -or
    $run.status -ne 'completed' -or $run.conclusion -ne 'success') {
    throw 'Required exact-source acceptance run did not pass'
}
$jobs = Invoke-RestMethod "$api/actions/runs/$($accepted.run)/attempts/$($run.run_attempt)/jobs?per_page=100" -Headers $headers
foreach ($name in @('catalog', 'experience', 'installer / installed')) {
    $matches = @($jobs.jobs | Where-Object { $_.name -ceq $name -and $_.conclusion -eq 'success' })
    if ($matches.Count -ne 1) { throw "Required acceptance job did not pass: $name" }
}
$artifact = Invoke-RestMethod "$api/actions/artifacts/$($accepted.artifact)" -Headers $headers
if ($artifact.name -cne 'prototype-0.1.44-server-tested' -or $artifact.expired -or
    $artifact.workflow_run.head_sha -cne $accepted.source -or
    [long]$artifact.workflow_run.id -ne [long]$accepted.run -or
    $artifact.digest -cne "sha256:$($accepted.archive_sha256)") {
    throw 'Accepted artifact identity mismatch'
}

$root = Join-Path ([IO.Path]::GetTempPath()) ("release44-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $root | Out-Null
$zip = Join-Path $root 'accepted.zip'
Invoke-WebRequest "$api/actions/artifacts/$($accepted.artifact)/zip" -Headers $headers -OutFile $zip
if ((Get-FileHash -LiteralPath $zip).Hash.ToLowerInvariant() -cne $accepted.archive_sha256) {
    throw 'Downloaded artifact ZIP hash mismatch'
}
$output = Join-Path $root 'assets'
Expand-Archive -LiteralPath $zip -DestinationPath $output
$expected = @(
    'dist/ImmichClone-0.1.44-prototype-x64-setup.exe',
    'dist/SHA256SUMS.txt',
    'docs/release-notes-0.1.44.md',
    'docs/validation-0.1.44.md'
)
if (@(Get-ChildItem -LiteralPath $output -Recurse -File -Force).Count -ne $expected.Count -or
    @($accepted.assets.PSObject.Properties).Count -ne $expected.Count) {
    throw 'Unexpected accepted archive or manifest members'
}
$files = @(foreach ($relative in $expected) {
    $path = Join-Path $output $relative
    $item = Get-Item -LiteralPath $path
    $hash = (Get-FileHash -LiteralPath $path).Hash.ToLowerInvariant()
    $pin = $accepted.assets.PSObject.Properties[$relative].Value
    if ($hash -cne $pin.sha256 -or $item.Length -ne [long]$pin.size) {
        throw "Accepted asset identity mismatch: $relative"
    }
    @{ Path = $path; Name = $item.Name; Length = $item.Length; Hash = $hash }
})
$installer = $files[0]
$checksum = (Get-Content -LiteralPath $files[1].Path -Raw).Trim()
if ($checksum -cnotmatch ('^' + $installer.Hash + '\s+\*?' + [regex]::Escape($installer.Name) + '$')) {
    throw 'The distributed checksum does not identify the tested installer'
}
if (!$Publish) {
    Write-Output "Verified accepted artifact $($accepted.artifact) and all four assets."
    return
}

$tags = @((Invoke-RestMethod "$api/git/matching-refs/tags/$tag" -Headers $headers) |
    Where-Object ref -CEQ "refs/tags/$tag")
if ($tags.Count -gt 1 -or ($tags.Count -eq 1 -and $tags[0].object.sha -cne $accepted.source)) {
    throw 'Existing release tag points elsewhere'
}
$all = @((Invoke-RestMethod "$api/releases?per_page=100" -Headers $headers) |
    Where-Object tag_name -CEQ $tag)
if ($all.Count -gt 1) { throw 'Release identity is ambiguous' }
if ($all.Count -eq 0) {
    if ($tags.Count -ne 0) { throw 'An existing release tag requires an existing matching draft' }
    $notes = Get-Content -LiteralPath $files[2].Path -Raw
    $notes += "`nExact tested source: $($accepted.source)`n`n"
    $notes += "Windows Server 2022 acceptance: https://github.com/$repo/actions/runs/$($accepted.run)`n`n"
    $notes += "Installer SHA256: $($installer.Hash)`n"
    $body = @{
        tag_name = $tag
        target_commitish = $accepted.source
        name = 'Immich Clone 0.1.44 Prototype'
        body = $notes
        draft = $true
        prerelease = $true
        make_latest = 'false'
    } | ConvertTo-Json
    $release = Invoke-RestMethod "$api/releases" -Method Post -Headers $headers -ContentType 'application/json' -Body $body
} else {
    $release = $all[0]
}
if (!$release.draft -or !$release.prerelease -or $release.target_commitish -cne $accepted.source) {
    throw 'Only a matching unpublished draft may be completed'
}
foreach ($asset in $release.assets) {
    $file = @($files | Where-Object Name -CEQ $asset.name)
    if ($file.Count -ne 1 -or $asset.state -ne 'uploaded' -or
        $asset.size -ne $file[0].Length -or $asset.digest -cne "sha256:$($file[0].Hash)") {
        throw 'Existing draft contains an unexpected asset'
    }
}
$uploadUrl = $release.upload_url -replace '\{.*$', ''
if ($uploadUrl -cne "https://uploads.github.com/repos/$repo/releases/$($release.id)/assets") {
    throw 'Unexpected GitHub asset destination'
}
foreach ($file in $files) {
    if ($release.assets.name -ccontains $file.Name) { continue }
    $name = [uri]::EscapeDataString($file.Name)
    $null = Invoke-RestMethod ("${uploadUrl}?name=$name") -Method Post -Headers $headers -ContentType 'application/octet-stream' -InFile $file.Path
}
$draft = Invoke-RestMethod "$api/releases/$($release.id)" -Headers $headers
Assert-ReleaseAssets $draft.assets
if (!$draft.draft -or !$draft.prerelease -or $draft.target_commitish -cne $accepted.source) {
    throw 'Draft identity changed during upload'
}
if ($tags.Count -eq 0) {
    $tagBody = @{ ref = "refs/tags/$tag"; sha = $accepted.source } | ConvertTo-Json
    $null = Invoke-RestMethod "$api/git/refs" -Method Post -Headers $headers -ContentType 'application/json' -Body $tagBody
}
$ref = Invoke-RestMethod "$api/git/ref/tags/$tag" -Headers $headers
if ($ref.object.type -cne 'commit' -or $ref.object.sha -cne $accepted.source) { throw 'Release tag identity mismatch' }
$body = @{ draft = $false; prerelease = $true; make_latest = 'false' } | ConvertTo-Json
$null = Invoke-RestMethod "$api/releases/$($draft.id)" -Method Patch -Headers $headers -ContentType 'application/json' -Body $body
$published = Invoke-RestMethod "$api/releases/tags/$tag" -Headers $headers
if ($published.id -ne $draft.id -or $published.draft -or !$published.prerelease) {
    throw 'Public release identity mismatch'
}
Assert-ReleaseAssets $published.assets
Write-Output $published.html_url
