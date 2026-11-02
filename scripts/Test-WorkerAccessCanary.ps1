# Disposable Windows-only proof of current authority, not a sandbox implementation.
$ErrorActionPreference = 'Stop'
$root = Join-Path $env:RUNNER_TEMP ('worker-canary-'+[guid]::NewGuid().ToString('N'))
$name = 'ImmichCanary'+[guid]::NewGuid().ToString('N').Substring(0,10)
$created = $false
$previousCGO = $env:CGO_ENABLED
function Native([string]$exe,[string[]]$arguments) { & $exe @arguments | Out-Null; if ($LASTEXITCODE -ne 0) { throw 'Synthetic canary fixture operation failed' } }
function Assert([bool]$value,[string]$label) { if (!$value) { throw $label }; Write-Output "PASS: $label" }
try {
 New-Item -ItemType Directory -Path $root | Out-Null
 Native icacls @($root,'/inheritance:r','/grant:r','*S-1-5-18:(OI)(CI)F','*S-1-5-32-544:(OI)(CI)F','*S-1-5-19:(OI)(CI)RX')
 foreach($dir in @('state','control','denied')) {New-Item -ItemType Directory -Path (Join-Path $root $dir) | Out-Null}
 foreach($dir in @('state','control')) {Native icacls @((Join-Path $root $dir),'/grant:r','*S-1-5-19:(OI)(CI)M')}
 Native icacls @((Join-Path $root 'denied'),'/inheritance:r','/grant:r','*S-1-5-18:(OI)(CI)F','*S-1-5-32-544:(OI)(CI)F')
 [IO.File]::WriteAllText((Join-Path $root 'state/secret.txt'),'synthetic-worker-secret')
 [IO.File]::WriteAllText((Join-Path $root 'state/catalog-canary.bin'),'synthetic-catalog-canary')
 [IO.File]::WriteAllText((Join-Path $root 'state/original-canary.bin'),'synthetic-original-before-probe')
 [IO.File]::WriteAllText((Join-Path $root 'denied/secret.txt'),'synthetic-worker-secret')
 $binary=Join-Path $root 'canary.exe'
 $env:CGO_ENABLED='0'
 Native go @('build','-trimpath','-o',$binary,'./experiments/worker-access-canary')
 $credential=New-Object PSCredential('NT AUTHORITY\LocalService',(ConvertTo-SecureString 'unused' -AsPlainText -Force))
 New-Service -Name $name -BinaryPathName ('"'+$binary+'" -mode service -service "'+$name+'" -root "'+$root+'"') -Credential $credential | Out-Null
 $created=$true
 Start-Service -Name $name
 $report=Join-Path $root 'control/result.json'
 for($i=0;$i -lt 100;$i++) {if(Test-Path -LiteralPath $report){break};Start-Sleep -Milliseconds 300}
 Assert (Test-Path -LiteralPath $report) 'LocalService canary completed within bounded wait'
 $result=Get-Content -LiteralPath $report -Raw | ConvertFrom-Json
 $expected=@('service_identity_localservice','unrestricted_identity_localservice','job_identity_localservice','unrestricted_secret_read_allowed','job_secret_read_allowed','unrestricted_catalog_read_allowed','job_catalog_read_allowed','unrestricted_original_write_allowed','job_original_write_allowed','unrestricted_private_read_denied','job_private_read_denied','unrestricted_private_write_denied','job_private_write_denied','unrestricted_large_allocation_allowed','job_large_allocation_refused','unrestricted_spawn_allowed','job_spawn_refused','job_peak_within_configured_bound','synthetic_original_write_verified','probe_complete')
 Assert (@($result.PSObject.Properties).Count -eq $expected.Count) 'canary report contains only expected boolean fields'
 foreach($key in $expected) {Assert ($result.$key -is [bool] -and $result.$key) $key}
 # A second admin-side check prevents a reported child success from substituting
 # for the actual synthetic file operation; values and paths are never emitted.
 Assert ([IO.File]::ReadAllText((Join-Path $root 'state/original-canary.bin')) -eq 'synthetic-child-write') 'admin verifies actual canary write'
 Assert ([IO.File]::ReadAllText((Join-Path $root 'denied/secret.txt')) -eq 'synthetic-worker-secret' -and !(Test-Path (Join-Path $root 'denied/forbidden.txt'))) 'private negative-control fixtures remain unchanged'
} finally {
 $env:CGO_ENABLED=$previousCGO
 if($created) {Stop-Service -Name $name -Force -ErrorAction SilentlyContinue;& sc.exe delete $name | Out-Null}
 if(Test-Path -LiteralPath $root){Remove-Item -LiteralPath $root -Recurse -Force}
}
