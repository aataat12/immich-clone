param([switch]$Release44)

# Keep the prior 0.1.43 combined acceptance runnable while exercising its
# full provider/recovery journey against the next exact release package.
$priorVersion = '0.1.42'
$candidateVersion = '0.1.43'
$rollbackProbeVersion = '0.1.44'
$expectedPriorHash = '18dbc670ce4d46809e2a90d5831d17b31f633050907290082f1e793294df8b87'
if ($Release44) {
    $priorVersion = '0.1.43'
    $candidateVersion = '0.1.44'
    $rollbackProbeVersion = $candidateVersion
    $expectedPriorHash = '96670e90c4abc55cdc07f480267c6f896b784a2a6c11de9f9f5333edb4158415'
}
$priorBuild = "$priorVersion-prototype"
$candidateBuild = "$candidateVersion-prototype"
$ErrorActionPreference = 'Stop'
$root = Join-Path $env:RUNNER_TEMP ('installer test ' + [guid]::NewGuid().ToString('N'))
if ($env:GITHUB_ACTIONS -ne 'true' -or (Get-Service ImmichClone -ErrorAction SilentlyContinue) -or (Test-Path 'HKLM:\SOFTWARE\ImmichClone')) { throw 'Installer acceptance requires an unregistered disposable CI host' }
$fixtureName = 'ImmichClone-install-'+[guid]::NewGuid().ToString('N')
$app = Join-Path $env:ProgramFiles $fixtureName
$initialState = Join-Path $env:ProgramData $fixtureName
$state = $initialState
if ((Test-Path -LiteralPath $app) -or (Test-Path -LiteralPath $initialState)) { throw 'Disposable default-layout names already exist' }
New-Item -ItemType Directory $root | Out-Null
if (!$env:IMMICH_PRIOR_INSTALLER) { throw 'Permanent published prior installer was not supplied' }
if (!(Test-Path -LiteralPath $env:IMMICH_PRIOR_INSTALLER)) { throw 'Published prior installer path is missing' }
$priorHash = (Get-FileHash -LiteralPath $env:IMMICH_PRIOR_INSTALLER).Hash.ToLowerInvariant()
if ($priorHash -ne $expectedPriorHash) { throw "Published $priorVersion installer hash mismatch" }
if (!$env:IMMICH_FULL_CONCURRENCY_SOURCE -or !$env:IMMICH_FULL_CONCURRENCY_RUN) { throw 'Full Setup concurrency handoff metadata is required' }
# Model a protected application location, not the runner temporary parent's
# permissive inherited code-write grants. Hostile permissions are exercised by
# the focused actual-Setup/standard-user gate before this combined transition.
$installerSid = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
& icacls $root /inheritance:r /grant:r '*S-1-5-18:(OI)(CI)F' '*S-1-5-32-544:(OI)(CI)F' ('*'+$installerSid+':(OI)(CI)F') '*S-1-5-19:(OI)(CI)RX' | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Protected installer fixture root failed' }
function Assert([bool]$condition,[string]$message) { if (!$condition) { throw $message }; Write-Output "PASS: $message" }
function BrowserPhase([string]$file,[string]$phase) {
    & node $file $phase
    Assert ($LASTEXITCODE -eq 0) ("installed browser acceptance: $file $phase")
}
function Get-OrganizationSnapshot($WebSession) {
    $url='http://127.0.0.1:2283/api/organization'
    $first=Invoke-RestMethod $url -WebSession $WebSession
    $assets=[Collections.Generic.List[object]]::new()
    $faces=[Collections.Generic.List[object]]::new()
    foreach($item in $first.assets){$assets.Add($item)}
    foreach($item in $first.faces){$faces.Add($item)}
    $cursor=$first.after;$pages=0
    while($cursor){
        if(++$pages -gt 64){throw 'Installed organization asset cursor did not terminate'}
        $page=Invoke-RestMethod ($url+'?after='+[Uri]::EscapeDataString($cursor)) -WebSession $WebSession
        foreach($item in $page.assets){$assets.Add($item)}
        $cursor=$page.after
    }
    $cursor=$first.face_after;$pages=0
    while($cursor){
        if(++$pages -gt 64){throw 'Installed organization face cursor did not terminate'}
        $page=Invoke-RestMethod ($url+'?face_after='+[Uri]::EscapeDataString($cursor)) -WebSession $WebSession
        foreach($item in $page.faces){$faces.Add($item)}
        $cursor=$page.face_after
    }
    $first.assets=$assets.ToArray();$first.faces=$faces.ToArray()
    return $first
}
function Install([string]$version) {
    if ($version -eq $priorVersion) { $setup=(Resolve-Path $env:IMMICH_PRIOR_INSTALLER).Path } else { $setup=(Resolve-Path "dist/ImmichClone-$version-prototype-x64-setup.exe").Path }
    $process = Start-Process $setup -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$app+'"'),('/STATE="'+$state+'"'),('/LOG="'+$root+'/setup-'+$version+'.log"')) -Wait -PassThru -WindowStyle Hidden
    if ($process.ExitCode -ne 0) {
        Get-Content "$root/setup-$version.log" -Tail 25 | ForEach-Object { Write-Host $_ }
        if (Test-Path "$app/install-diagnostic.txt") { Get-Content "$app/install-diagnostic.txt" | ForEach-Object { Write-Host $_ } }
    }
    return $process.ExitCode
}
try {
    Assert ((Install $priorVersion) -eq 0) 'installer fresh install succeeds in path containing spaces'
    $previousBinary=Join-Path $root "actual-$priorVersion.exe"
    Copy-Item "$app/immichclone.exe" $previousBinary
    Assert ((& $previousBinary -mode version) -eq $priorBuild) 'older-writer probe uses the actual previously released executable'
    Assert (Test-Path "$app/media/ffmpeg-8.1.3-7eb77d88e2e9/ffmpeg-8.1.3-source.tar.gz") 'installer includes exact media source and notices'
    $service = Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
    Write-Output ('Disposable CI service evidence: '+($service | Select-Object Name,StartName,StartMode,State | ConvertTo-Json -Compress))
    if (!$service -or $service.StartName -ne 'NT AUTHORITY\LocalService' -or $service.StartMode -ne 'Auto') { Get-Content "$root/setup-$priorVersion.log" -Tail 35 }
    Assert (Test-Path "$app/Export-Diagnostics.ps1") 'installer includes standalone diagnostic exporter'
    Assert ($service.StartName -eq 'NT AUTHORITY\LocalService' -and $service.StartMode -eq 'Auto') 'installer configures restricted automatic service'
    & "$env:SystemRoot/System32/WindowsPowerShell/v1.0/powershell.exe" -NoProfile -NonInteractive -ExecutionPolicy Bypass -File scripts/Test-LocalPrototype.ps1
    Assert ($LASTEXITCODE -eq 0) 'local acceptance helper executes under built-in Windows PowerShell'
    $session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $code = (Get-Content "$state/access-code.txt" -Raw).Trim()
    $null = Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $session
    $fixture = Join-Path $root 'synthetic.txt'
    $privateMarker = 'private-package-canary-'+[guid]::NewGuid().ToString('N')
    [IO.File]::WriteAllText($fixture,$privateMarker)
    $asset = (Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=synthetic.txt' -Method Post -Headers @{'X-Immich-Request'='1'} -InFile $fixture -WebSession $session).asset
    $hash = (Get-FileHash $fixture).Hash
    $beforeAI=Join-Path $root 'before-upgrade-ai.png'
    Invoke-WebRequest 'https://raw.githubusercontent.com/scikit-image/scikit-image/e8a42ba85aaf5fd9322ef9ca51bc21063b22fcae/skimage/data/astronaut.png' -OutFile $beforeAI
    Assert ((Get-FileHash $beforeAI).Hash.ToLowerInvariant() -eq '88431cd9653ccd539741b555fb0a46b61558b301d4110412b5bc28b5e3ea6cb5') 'pre-upgrade public fixture hash verified'
    $beforeAsset=(Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=before-upgrade-ai.png' -Method Post -Headers @{'X-Immich-Request'='1'} -InFile $beforeAI -WebSession $session).asset
    $null=Invoke-RestMethod "http://127.0.0.1:2283/api/assets/$($beforeAsset.id)/analyze" -Method Post -Headers @{'X-Immich-Request'='1'} -WebSession $session
    for($i=0;$i -lt 30;$i++){ $beforeOrg=Get-OrganizationSnapshot $session;$beforeEntry=$beforeOrg.assets | Where-Object {$_.asset.id -eq $beforeAsset.id};if($beforeEntry.analysis.status -eq 'ready'){break};Start-Sleep -Seconds 1 }
    Assert ($beforeEntry.analysis.status -eq 'ready') 'actual previous installer supplies ready AI data'
    $beforeFace=@($beforeOrg.faces | Where-Object asset_id -eq $beforeAsset.id)[0]
    $null=Invoke-RestMethod 'http://127.0.0.1:2283/api/faces/label' -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{ids=@($beforeFace.id);name='Before upgrade person'}|ConvertTo-Json) -WebSession $session
    $null=Invoke-RestMethod "http://127.0.0.1:2283/api/assets/$($beforeAsset.id)/tags" -Method Post -Headers @{'X-Immich-Request'='1'} -Body '{"tags":["Before upgrade tag"]}' -WebSession $session

    # Fail the first initial upgrade upgrade too, then prove old executable/provider
    # and original data still work before attempting a successful upgrade.
    Stop-Service ImmichClone
    & sc.exe config ImmichClone start= disabled | Out-Null
    Assert ($LASTEXITCODE -eq 0) 'disposable previous service is disabled before corrected recovery'
    $disabled = Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
    Assert ($disabled.State -eq 'Stopped' -and $disabled.StartMode -eq 'Disabled') 'upgrade fixture reproduces retained prior build with stopped disabled service'
    Assert ((& $previousBinary -mode version) -eq $priorBuild) 'disabled-service recovery starts with the actual previous release'
    $env:IMMICH_INSTALL_TEST_FAILURE='1'
    $firstUpgradeFailure=Install $candidateVersion
    Remove-Item Env:IMMICH_INSTALL_TEST_FAILURE
    Assert ($firstUpgradeFailure -ne 0) 'initial upgrade failed upgrade is rejected'
    Assert ((Invoke-RestMethod http://127.0.0.1:2283/health).build -eq $priorBuild) 'initial upgrade rollback restores prior build'
    Assert ((Get-FileHash "$app/ai/opencv-4.12.0-nr01-zlib-1.3.2-15d1c2ca7ef3/ai-worker.exe").Hash.ToLowerInvariant() -eq '15d1c2ca7ef3f526b34ac4823f9337c51280ad171ddf43df4f3af5d8128a7c58') 'initial upgrade rollback retains old immutable worker'
    Assert ((Get-FileHash "$state/originals/$($asset.id)").Hash -eq $hash) 'initial upgrade rollback preserves original bytes'
    Stop-Service ImmichClone
    & sc.exe config ImmichClone start= disabled | Out-Null
    Assert ($LASTEXITCODE -eq 0) 'prior service disabled again before successful corrected upgrade'
    $disabled = Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
    Assert ($disabled.State -eq 'Stopped' -and $disabled.StartMode -eq 'Disabled') 'successful corrected upgrade begins with stopped disabled prior service'
    Assert ((Install $candidateVersion) -eq 0) 'installer upgrade succeeds'
    $recoveredService = Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
    Assert ($recoveredService.State -eq 'Running' -and $recoveredService.StartMode -eq 'Auto') 'corrected upgrade recovers automatic running service from disabled prior installation'
    Assert ((& "$app/immichclone.exe" -mode version) -eq $candidateBuild) 'upgrade deploys new executable version'
    Assert ((Invoke-RestMethod http://127.0.0.1:2283/health).build -eq $candidateBuild) 'installed health reports actual executable build'
    Assert ((Get-FileHash "$state/originals/$($asset.id)").Hash -eq $hash) 'upgrade preserves original bytes'
    $session=[Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $null=Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $session
    $preservedOrg=Get-OrganizationSnapshot $session
    $preservedEntry=$preservedOrg.assets | Where-Object {$_.asset.id -eq $beforeAsset.id}
    Assert (@($preservedOrg.faces | Where-Object name -eq 'Before upgrade person').Count -eq 1 -and $preservedEntry.tags -contains 'Before upgrade tag' -and $preservedEntry.analysis.status -eq 'ready' -and $preservedEntry.analysis.provider -eq 'opencv-4.12.0-nr01-zlib-1.3.2-15d1c2ca7ef3-bgr960-v1') 'upgrade preserves old names tags and ready analysis without rerunning it'

    # Scan installed distribution payloads, never the protected runtime rollback
    # snapshot (which intentionally preserves the catalog and access code).
    $forbidden = Join-Path $root 'private-package-canaries.txt'
    [IO.File]::WriteAllText($forbidden,($code+"`n"+$privateMarker+"`n"),[Text.UTF8Encoding]::new($false))
    & python scripts/Check-ArtifactPrivacy.py --forbidden-file $forbidden "$app/immichclone.exe" "$app/installer-files.exe" "$app/Export-Diagnostics.ps1" "$app/Manage-Service.ps1" "$app/THIRD-PARTY-LICENSES.txt" "$app/prototype-notes.txt" "$app/media" "$app/ai"
    Assert ($LASTEXITCODE -eq 0) 'installed payload omits runtime access code and private media canary'

    $session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $null = Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $session
    $heic = Join-Path $root 'synthetic.heic'
    [IO.File]::WriteAllBytes($heic,[Convert]::FromBase64String((Get-Content experiments/media-worker/fixtures/synthetic.heic.b64 -Raw)))
    $heicAsset = (Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=synthetic.heic' -Method Post -Headers @{'X-Immich-Request'='1'} -InFile $heic -WebSession $session).asset
    $heicHash = (Get-FileHash $heic).Hash
    for ($i=0;$i -lt 20;$i++) {
        $photo = (Invoke-RestMethod http://127.0.0.1:2283/api/assets -WebSession $session) | Where-Object id -eq $heicAsset.id
        if ($photo.preview -ne 'queued') { break }
        Start-Sleep -Seconds 1
    }
    Assert ($photo.preview -eq 'ready' -and $photo.width -eq 160 -and $photo.height -eq 120) 'installed restricted service executes bundled HEIC provider from a path containing spaces'
    $aiFixture=Join-Path $root 'public-ai.png'
    Invoke-WebRequest 'https://raw.githubusercontent.com/scikit-image/scikit-image/e8a42ba85aaf5fd9322ef9ca51bc21063b22fcae/skimage/data/astronaut.png' -OutFile $aiFixture
    Assert ((Get-FileHash $aiFixture).Hash.ToLowerInvariant() -eq '88431cd9653ccd539741b555fb0a46b61558b301d4110412b5bc28b5e3ea6cb5') 'public AI fixture hash verified'
    # Keep the same public image pixels but a distinct original identity so this
    # job must actually execute the replacement provider instead of reusing ready data.
    [IO.File]::WriteAllBytes($aiFixture,([IO.File]::ReadAllBytes($aiFixture)+[Text.Encoding]::ASCII.GetBytes('new provider fixture')))

    $aiHeaders=@{'X-Immich-Request'='1'}
    $aiAsset=(Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=public-ai.png' -Method Post -Headers $aiHeaders -InFile $aiFixture -WebSession $session).asset
    $null=Invoke-RestMethod "http://127.0.0.1:2283/api/assets/$($aiAsset.id)/analyze" -Method Post -Headers $aiHeaders -WebSession $session
    for($i=0;$i -lt 30;$i++){ $org=Get-OrganizationSnapshot $session;$entry=$org.assets | Where-Object {$_.asset.id -eq $aiAsset.id};if($entry.analysis.status -ne 'queued'){break};Start-Sleep -Seconds 1 }
    Assert ($entry.analysis.status -eq 'ready') 'installed service executes bundled native AI without SDK Python or GPU'
    $face=@($org.faces | Where-Object asset_id -eq $aiAsset.id)[0]
    $null=Invoke-RestMethod 'http://127.0.0.1:2283/api/faces/label' -Method Post -Headers $aiHeaders -Body (@{ids=@($face.id);name='Installer Person'}|ConvertTo-Json) -WebSession $session
    $null=Invoke-RestMethod "http://127.0.0.1:2283/api/assets/$($aiAsset.id)/tags" -Method Post -Headers $aiHeaders -Body '{"tags":["Installer tag"]}' -WebSession $session
    # Exercise6A through the installed executable and installed immutable providers.
    $env:IMMICH_TEST_STATE=$state
    & node tests/media-ai.cjs start
    Assert ($LASTEXITCODE -eq 0) 'installed6A HEIC analysis and paused video controls pass'
    Restart-Service ImmichClone
    & node tests/media-ai.cjs finish
    Assert ($LASTEXITCODE -eq 0) 'installed6A timestamped video inference and original checks pass'
    & node tests/reanalysis.cjs prepare
    Assert ($LASTEXITCODE -eq 0) 'installed reanalysis HEIC discard apply and paused video pass'
    Restart-Service ImmichClone
    & node tests/reanalysis.cjs finish
    Assert ($LASTEXITCODE -eq 0) 'installed reanalysis video restart discard apply and narrow layout pass'
    BrowserPhase 'tests/hidden-media.cjs' 'prepare'
    Restart-Service ImmichClone
    BrowserPhase 'tests/hidden-media.cjs' 'finish'
    $env:IMMICH_NAV_ATTACH='1'
    try { & node tests/hidden-navigation.cjs;Assert ($LASTEXITCODE -eq 0) 'installed Show/Hide Hidden Media toggle and selector navigation pass'; & node tests/hidden-batch.cjs;Assert ($LASTEXITCODE -eq 0) 'installed multi-page Hidden Media batch actions pass' } finally { Remove-Item Env:IMMICH_NAV_ATTACH -ErrorAction SilentlyContinue }
    $env:IMMICH_IMPORT_SOURCE_FIXTURES=Join-Path $root 'import source'
    & node tests/import-source-loss.cjs
    Assert ($LASTEXITCODE -eq 0) 'installed import survives real source loss and rejects changed receipt replay'
    Remove-Item Env:IMMICH_IMPORT_SOURCE_FIXTURES
    BrowserPhase 'tests/backup.cjs' ''
    BrowserPhase 'tests/backup.cjs' '--discard'
    $env:IMMICH_COMPAT_INSTALLED='1'
    BrowserPhase 'tests/media-compatibility-installed.cjs' ''
    # Retain an unapplied proposal through failed upgrade and independent restore.
    $proposalReceipt=Get-Content (Join-Path $env:IMMICH_6A_FIXTURES 'app-receipt.json') -Raw | ConvertFrom-Json
    $proposalID=$proposalReceipt.'face.heic'
    $session=[Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $null=Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers $aiHeaders -Body (@{code=$code}|ConvertTo-Json) -WebSession $session
    $null=Invoke-RestMethod "http://127.0.0.1:2283/api/assets/$proposalID/reanalysis" -Method Post -Headers $aiHeaders -Body '{"action":"start"}' -WebSession $session
    for($i=0;$i -lt 30;$i++){ $proposalOrg=Get-OrganizationSnapshot $session;$proposal=$proposalOrg.assets | Where-Object {$_.asset.id -eq $proposalID};if($proposal.reanalysis.status -eq 'ready'){break};Start-Sleep -Seconds 1 }
    Assert ($proposal.reanalysis.status -eq 'ready') 'installed service prepares a durable unapplied proposal'
    $proposalFaceIDs=@($proposalOrg.faces | Where-Object asset_id -eq $proposalID | Sort-Object id | ForEach-Object id) -join ','
    # A non-regular disposable preview blocks cleanup for both LocalService and
    # installer initialization. No production permissions are changed.
    BrowserPhase 'tests/deletion.cjs' 'prepare-denied'
    $deleteID=(Get-Content (Join-Path $env:IMMICH_6A_FIXTURES 'deletion-receipt.json') -Raw | ConvertFrom-Json).id
    $deletionArchive=Join-Path $root 'before-installed-deletion.zip'
    Invoke-WebRequest http://127.0.0.1:2283/api/backup -WebSession $session -OutFile $deletionArchive
    $deleteBlock=Join-Path $state ('previews/'+$deleteID+'.png')
    Assert (!(Test-Path $deleteBlock)) 'unsupported disposable deletion fixture has no preview'
    New-Item -ItemType Directory $deleteBlock | Out-Null
    BrowserPhase 'tests/deletion.cjs' 'denied'
    $beforeRollbackHash = (Get-FileHash -LiteralPath "$app/immichclone.exe").Hash
    $env:IMMICH_INSTALL_TEST_FAILURE = '1'
    $failedCode = Install $rollbackProbeVersion
    Remove-Item Env:IMMICH_INSTALL_TEST_FAILURE
    $installLog = Get-Content (Join-Path $env:ProgramData 'ImmichCloneDiagnostics/installer.jsonl') -Raw
    Assert ($installLog.Contains('installer_failed') -and $installLog.Contains('rollback_complete') -and !$installLog.Contains($state)) 'installer logs failed-upgrade and rollback stages without data paths'
    Assert ($failedCode -ne 0) 'injected failed upgrade returns failure'
    Assert ((& "$app/immichclone.exe" -mode version) -eq $candidateBuild -and (Get-FileHash -LiteralPath "$app/immichclone.exe").Hash -eq $beforeRollbackHash) 'failed upgrade restores exact previous executable'
    Assert ((Invoke-RestMethod http://127.0.0.1:2283/health).status -eq 'ok') 'rollback restores healthy service'
    BrowserPhase 'tests/deletion.cjs' 'pending'
    Assert ((Get-FileHash "$state/originals/$($asset.id)").Hash -eq $hash) 'failed upgrade preserves original bytes'
    Assert ((Get-FileHash "$state/originals/$($heicAsset.id)").Hash -eq $heicHash) 'failed upgrade preserves HEIC original'
    $session=[Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $null=Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers $aiHeaders -Body (@{code=$code}|ConvertTo-Json) -WebSession $session
    $org=Get-OrganizationSnapshot $session
    Assert (@($org.faces | Where-Object name -eq 'Installer Person').Count -eq 1) 'failed-upgrade rollback preserves person assignments'
    $sixAReceipt=Get-Content (Join-Path $env:IMMICH_6A_FIXTURES 'app-receipt.json') -Raw | ConvertFrom-Json
    $sixA=$org.assets | Where-Object { $_.asset.id -eq $sixAReceipt.'video.mp4' }
    Assert ($sixA.analysis.video.next -eq 12 -and @($org.faces | Where-Object name -eq 'Video Test Person').Count -eq 1) 'failed upgrade preserves6A observations and manual corrections'
    $proposal=$org.assets | Where-Object {$_.asset.id -eq $proposalID}
    Assert ($proposal.reanalysis.status -eq 'ready' -and $proposal.analysis.status -eq 'ready') 'failed upgrade retains unapplied proposal and completed results'
    # The actual previous executable understands deletion intents.
    # It must retain the pending original and refuse reimport until cleanup.
    # The unaware 0.1.33 writer/conflict proof remains in 0.1.34 evidence.
    $legacyWriterCatalog=Invoke-RestMethod http://127.0.0.1:2283/api/export -WebSession $session
    Assert ($legacyWriterCatalog.catalog_schema -eq "1") 'published old-writer deletion control uses an unpromoted legacy catalog; automatic schema-two refusal is tested separately'
    Stop-Service ImmichClone
    & $previousBinary -mode verify -data $state
    Assert ($LASTEXITCODE -eq 0) 'actual previous executable verifies catalog containing a pending intent'
    # The previous version refuses elevated direct serve; exercise the real binary
    # under the existing LocalService registration on this disposable host.
    $currentBinary=Join-Path $root 'current-before-older-writer.exe'
    Copy-Item "$app/immichclone.exe" $currentBinary
    try {
        Copy-Item $previousBinary "$app/immichclone.exe" -Force
        Assert ((Get-FileHash "$app/immichclone.exe").Hash -eq (Get-FileHash $previousBinary).Hash) 'older-writer service uses exact published previous executable'
        Start-Service ImmichClone
        $oldReady=$false
        for($i=0;$i -lt 40;$i++) {
            try { if((Invoke-RestMethod http://127.0.0.1:2283/health -TimeoutSec 2).build -eq $priorBuild){$oldReady=$true;break} } catch {}
            Start-Sleep -Milliseconds 250
        }
        Assert $oldReady 'actual previous executable serves the pending-intent catalog'
        $oldSession=[Microsoft.PowerShell.Commands.WebRequestSession]::new()
        $null=Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers $aiHeaders -Body (@{code=$code}|ConvertTo-Json) -WebSession $oldSession
        $oldSource=Join-Path $root 'older-writer-source.txt'
        [IO.File]::WriteAllText($oldSource,'disposable deletion permission fixture',[Text.UTF8Encoding]::new($false))
        $oldImport=Invoke-WebRequest 'http://127.0.0.1:2283/api/upload?name=older-writer-reimport.txt' -Method Post -Headers $aiHeaders -InFile $oldSource -WebSession $oldSession -SkipHttpErrorCheck
        Assert ($oldImport.StatusCode -eq 500 -and $oldImport.Content.Contains('no success recorded')) 'actual previous aware writer refuses reimport while deletion is pending'
        $oldPending=Invoke-RestMethod http://127.0.0.1:2283/api/deletions -WebSession $oldSession
        $oldAssets=Invoke-RestMethod http://127.0.0.1:2283/api/assets -WebSession $oldSession
        Assert (@($oldPending.pending | Where-Object {$_.id -eq $deleteID -and $_.status -eq 'pending'}).Count -eq 1 -and @($oldAssets | Where-Object id -eq $deleteID).Count -eq 0) 'previous writer preserves the intent without republishing the removed catalog item'
        Assert ((Get-FileHash "$state/originals/$deleteID").Hash -eq (Get-FileHash $oldSource).Hash) 'rejected previous-writer import preserves pending original bytes'
    } finally {
        Stop-Service ImmichClone -Force -ErrorAction SilentlyContinue
        Copy-Item $currentBinary "$app/immichclone.exe" -Force
        Assert ((Get-FileHash "$app/immichclone.exe").Hash -eq (Get-FileHash $currentBinary).Hash) 'older-writer fixture restores exact candidate executable'
        Start-Service ImmichClone
    }
    Remove-Item -LiteralPath $deleteBlock
    BrowserPhase 'tests/deletion.cjs' 'retry'
    Remove-Item Env:IMMICH_TEST_STATE
    # Restore the actual pre-deletion archive independently while the installed
    # service remains online. Its old state is never used as a repair source.
    $deleteRestored=Join-Path $root 'pre-deletion restored'
    & "$app/immichclone.exe" -mode restore -data $deleteRestored -archive $deletionArchive
    Assert ($LASTEXITCODE -eq 0) 'installed executable restores the actual pre-deletion archive independently'
    $env:IMMICH_INSTALLED_DELETE_RESTORE=$deleteRestored
    & go test -count=1 -timeout=120s ./internal/library -run '^TestInstalledDeletionArchiveRestored$'
    Assert ($LASTEXITCODE -eq 0) 'independent pre-deletion restore retains hidden original and preexisting named faces'
    Remove-Item Env:IMMICH_INSTALLED_DELETE_RESTORE
    $recoveryTool = Join-Path $root 'recovery-tool.exe'
    Copy-Item "$app/immichclone.exe" $recoveryTool
    $session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $null = Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $session
    $archive = Join-Path $root 'recovery.zip'
    Invoke-WebRequest http://127.0.0.1:2283/api/backup -WebSession $session -OutFile $archive
    $uninstall = Start-Process "$app/unins000.exe" -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru -WindowStyle Hidden
    Assert ($uninstall.ExitCode -eq 0) 'silent uninstall succeeds'
    Assert (!(Get-Service ImmichClone -ErrorAction SilentlyContinue)) 'uninstall removes service'
    Assert ((Get-FileHash "$state/originals/$($asset.id)").Hash -eq $hash) 'uninstall retains original bytes'
    Assert ((Test-Path "$state/catalog.db") -and (Test-Path "$state/access-code.txt")) 'uninstall retains catalog and access code'
    # Make the old installation unavailable without destroying its files.
    if ([IO.Path]::GetFullPath($state) -ne [IO.Path]::GetFullPath($initialState) -or (Split-Path $state -Leaf) -ne $fixtureName) { throw 'Offline fixture target escaped disposable default state' }
    Move-Item -LiteralPath $state -Destination ($state+'.offline')
    $state = Join-Path $root 'restored state'
    & $recoveryTool -mode restore -data $state -archive $archive
    Assert ($LASTEXITCODE -eq 0) 'offline restoration succeeds at a different location'
    Assert ((Install $candidateVersion) -eq 0) 'installer starts a restored library without the original data folder'
    $session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $restoredCode = (Get-Content "$state/access-code.txt" -Raw).Trim()
    $null = Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$restoredCode}|ConvertTo-Json) -WebSession $session
    $retrieved = Join-Path $root 'restored-download.txt'
    Invoke-WebRequest "http://127.0.0.1:2283/api/assets/$($asset.id)/original" -WebSession $session -OutFile $retrieved
    Assert ((Get-FileHash $retrieved).Hash -eq $hash) 'reinstalled service authenticates with restored secret and returns original bytes'
    for ($i=0;$i -lt 20;$i++) {
        $photo = (Invoke-RestMethod http://127.0.0.1:2283/api/assets -WebSession $session) | Where-Object id -eq $heicAsset.id
        if ($photo.preview -eq 'ready') { break }
        Start-Sleep -Seconds 1
    }
    Assert ($photo.preview -eq 'ready' -and (Get-FileHash "$state/originals/$($heicAsset.id)").Hash -eq $heicHash) 'independent reinstall regenerates HEIC preview and preserves original'

    $org=Get-OrganizationSnapshot $session
    Assert (@($org.faces | Where-Object name -eq 'Installer Person').Count -eq 1) 'restored installer retains named faces'
    $sixA=$org.assets | Where-Object { $_.asset.id -eq $sixAReceipt.'video.mp4' }
    Assert ($sixA.analysis.video.next -eq 12 -and @($org.faces | Where-Object name -eq 'Video Test Person').Count -eq 1) 'independent installed restore retains6A observations and corrections'
    $proposal=$org.assets | Where-Object {$_.asset.id -eq $proposalID}
    Assert ($proposal.reanalysis.status -eq 'ready' -and $proposal.analysis.status -eq 'ready') 'independent installed restore retains unapplied proposal and completed results'
    $null=Invoke-RestMethod "http://127.0.0.1:2283/api/assets/$proposalID/reanalysis" -Method Post -Headers $aiHeaders -Body '{"action":"apply"}' -WebSession $session
    $appliedOrg=Get-OrganizationSnapshot $session
    $appliedProposal=$appliedOrg.assets | Where-Object {$_.asset.id -eq $proposalID}
    $restoredFaceIDs=@($appliedOrg.faces | Where-Object asset_id -eq $proposalID | Sort-Object id | ForEach-Object id) -join ','
    Assert ($appliedProposal.reanalysis.status -eq 'applied' -and $appliedProposal.reanalysis.added -eq 0 -and $restoredFaceIDs -eq $proposalFaceIDs) 'independently restored proposal applies without replacing or duplicating existing faces'
    $videoFace=@($org.faces | Where-Object name -eq 'Video Test Person')[0]
    Invoke-WebRequest ("http://127.0.0.1:2283/api/faces/"+$videoFace.id+"/context") -WebSession $session -OutFile "$root/restored-frame.png"
    Assert ((Get-Item "$root/restored-frame.png").Length -gt 0) 'restored installed provider regenerates sampled frame context'
    $crop=Join-Path $root 'restored-face.png'
    Invoke-WebRequest ("http://127.0.0.1:2283/api/faces/"+$face.id+"/crop") -WebSession $session -OutFile $crop
    $aiRestored=$org.assets | Where-Object {$_.asset.id -eq $aiAsset.id}
    Assert ((Get-Item $crop).Length -gt 0 -and $aiRestored.tags -contains 'Installer tag') 'restored face cache regenerates and custom tags survive'
    $env:IMMICH_TEST_STATE=$state
    BrowserPhase 'tests/media-compatibility-installed.cjs' '--restored'
    Remove-Item Env:IMMICH_TEST_STATE
    Remove-Item Env:IMMICH_COMPAT_INSTALLED
    $uninstall = Start-Process "$app/unins000.exe" -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru -WindowStyle Hidden
    Assert ($uninstall.ExitCode -eq 0) 'restored installation uninstalls cleanly'
} finally {
    Remove-Item Env:IMMICH_INSTALL_TEST_FAILURE -ErrorAction SilentlyContinue
    Remove-Item Env:IMMICH_COMPAT_INSTALLED -ErrorAction SilentlyContinue
    if (Get-Service ImmichClone -ErrorAction SilentlyContinue) { Stop-Service ImmichClone -Force -ErrorAction Continue; & sc.exe delete ImmichClone }
    # Delete only the exact GUID names created by this disposable fixture.
    foreach ($target in @($app,($app+'-upgrade-recovery'),$initialState,($initialState+'.offline'),$root)) {
        $absolute = [IO.Path]::GetFullPath($target)
        $allowed = @((Join-Path $env:ProgramFiles $fixtureName),(Join-Path $env:ProgramFiles ($fixtureName+'-upgrade-recovery')),(Join-Path $env:ProgramData $fixtureName),(Join-Path $env:ProgramData ($fixtureName+'.offline')),$root)
        if ($absolute -notin $allowed) { throw 'Cleanup target escaped disposable installer fixture' }
        if (Test-Path -LiteralPath $absolute) { Remove-Item -LiteralPath $absolute -Recurse -Force -ErrorAction Continue }
    }
}
