# Disposable Windows Server candidate proof. This is not production integration.
$ErrorActionPreference='Stop'
$root=Join-Path $env:RUNNER_TEMP ('worker-isolation-'+[guid]::NewGuid().ToString('N'))
$name='ImmichIsolation'+[guid]::NewGuid().ToString('N').Substring(0,10); $created=$false
function Native([string]$exe,[string[]]$a){& $exe @a|Out-Null;if($LASTEXITCODE-ne 0){throw "fixture operation failed: $exe"}}
function Assert([bool]$v,[string]$m){if(!$v){throw $m};Write-Output "PASS: $m"}
try {
 New-Item -ItemType Directory -Path $root,(Join-Path $root 'state'),(Join-Path $root 'input'),(Join-Path $root 'scratch'),(Join-Path $root 'denied'),(Join-Path $root 'control')|Out-Null
 [IO.File]::WriteAllText((Join-Path $root 'state/secret.txt'),'synthetic-worker-secret')
 [IO.File]::WriteAllText((Join-Path $root 'state/catalog-canary.bin'),'synthetic-catalog-canary')
 [IO.File]::WriteAllText((Join-Path $root 'state/original-canary.bin'),'synthetic-original-before-probe')
 [IO.File]::WriteAllText((Join-Path $root 'input/required.txt'),'synthetic-required-input')
 Native icacls @($root,'/inheritance:r','/grant:r','*S-1-5-18:(OI)(CI)F','*S-1-5-32-544:(OI)(CI)F','*S-1-5-19:(OI)(CI)RX')
 Native icacls @((Join-Path $root 'state'),'/grant:r','*S-1-5-19:(OI)(CI)M')
 Native icacls @((Join-Path $root 'control'),'/grant:r','*S-1-5-19:(OI)(CI)M')
 Native icacls @((Join-Path $root 'denied'),'/inheritance:r','/grant:r','*S-1-5-18:(OI)(CI)F','*S-1-5-32-544:(OI)(CI)F')
 Native go @('build','-trimpath','-o',(Join-Path $root 'canary.exe'),'./experiments/worker-access-canary')
 Native icacls @($root,'/grant','*S-1-5-12:(OI)(CI)RX')
 Native icacls @((Join-Path $root 'input'),'/grant','*S-1-5-12:(OI)(CI)RX')
 Native icacls @((Join-Path $root 'scratch'),'/grant','*S-1-5-12:(OI)(CI)M')
 Native icacls @((Join-Path $root 'state'),'/deny','*S-1-5-12:(OI)(CI)F')
 Native icacls @((Join-Path $root 'denied'),'/deny','*S-1-5-12:(OI)(CI)F')
 $cred=New-Object PSCredential('NT AUTHORITY\LocalService',(ConvertTo-SecureString 'unused' -AsPlainText -Force))
 New-Service -Name $name -BinaryPathName ('"'+(Join-Path $root 'canary.exe')+'" -mode candidate-service -service "'+$name+'" -root "'+$root+'"') -Credential $cred|Out-Null;$created=$true;Start-Service $name
	$report=Join-Path $root 'control/result.json';for($i=0;$i-lt 100;$i++){if(Test-Path $report){break};Start-Sleep -Milliseconds 300};if(!(Test-Path $report)){Write-Output ('service-started='+[bool](Test-Path (Join-Path $root 'control/service-started')));& sc.exe query $name;throw 'restricted candidate completed'}
 $r=Get-Content $report -Raw|ConvertFrom-Json
 Write-Output ('Candidate report: '+($r|ConvertTo-Json -Compress))
 Assert($r.probe_complete) 'restricted launcher completed'
 foreach($k in @('service_identity_localservice','restricted_identity_localservice','restricted_input_read_allowed','restricted_scratch_write_allowed','restricted_state_secret_denied','restricted_catalog_denied','restricted_original_write_denied','restricted_private_read_denied','restricted_process_spawn_denied','job_peak_within_configured_bound')){Assert($r.$k) $k}
 Assert((Get-Content (Join-Path $root 'state/original-canary.bin') -Raw) -eq 'synthetic-original-before-probe') 'original canary preserved'
} finally {if($created){Stop-Service $name -Force -ErrorAction SilentlyContinue;& sc.exe delete $name|Out-Null};if(Test-Path $root){Remove-Item $root -Recurse -Force}}

