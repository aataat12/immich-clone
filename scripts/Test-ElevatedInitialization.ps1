param([Parameter(Mandatory=$true)][string]$Binary)
$ErrorActionPreference = 'Stop'
function Assert([bool]$Condition,[string]$Label) { if (!$Condition) { throw $Label }; Write-Host "PASS: $Label" }
Add-Type @'
using System;
using System.Runtime.InteropServices;
public static class InitElevationProbe {
 [DllImport("kernel32.dll")] static extern IntPtr GetCurrentProcess();
 [DllImport("kernel32.dll")] static extern bool CloseHandle(IntPtr h);
 [DllImport("advapi32.dll", SetLastError=true)] static extern bool OpenProcessToken(IntPtr process,uint access,out IntPtr token);
 [DllImport("advapi32.dll", SetLastError=true)] static extern bool GetTokenInformation(IntPtr token,int kind,out int elevation,int length,out int returned);
 public static bool Elevated() {
  IntPtr token;
  if(!OpenProcessToken(GetCurrentProcess(),8,out token)) throw new System.ComponentModel.Win32Exception(Marshal.GetLastWin32Error());
  try {int elevation,returned;if(!GetTokenInformation(token,20,out elevation,4,out returned)) throw new System.ComponentModel.Win32Exception(Marshal.GetLastWin32Error());return elevation!=0;}
  finally {CloseHandle(token);}
 }
}
'@
$root=Join-Path $env:RUNNER_TEMP ('elevated-init-'+[guid]::NewGuid().ToString('N'))
$junctions=@()
$script:attempt=0
$refusal='Elevated initialization is refused; use the installer to initialize under LocalService.'
function Snapshot([string]$Path) {
 $parts=@()
 $items=@((Get-Item -LiteralPath $Path -Force)) + @(Get-ChildItem -LiteralPath $Path -Force -Recurse)
 foreach($item in ($items | Sort-Object FullName)) {
  if(($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0){throw 'Snapshot fixture unexpectedly contains a reparse point'}
  $part=$item.FullName+'|'+(Get-Acl -LiteralPath $item.FullName).Sddl
  if(!$item.PSIsContainer){$part+='|'+(Get-FileHash -LiteralPath $item.FullName -Algorithm SHA256).Hash}
  $parts+=$part
 }
 return $parts -join "`n"
}
function Run-GuardedMode([string]$Mode,[string]$State,[string]$Logs) {
 $script:attempt++
 $out=Join-Path $root ('output-'+$script:attempt+'.txt')
 $err=Join-Path $root ('error-'+$script:attempt+'.txt')
 $process=Start-Process -FilePath $Binary -ArgumentList @('-mode',$Mode,'-data',('"'+$State+'"'),'-logs',('"'+$Logs+'"')) -PassThru -RedirectStandardOutput $out -RedirectStandardError $err
 # Windows PowerShell can discard the exit status of a fast Start-Process
 # child unless its native process handle is retained before waiting.
 $null=$process.Handle
 try {
  if(!$process.WaitForExit(10000)) { $process.Kill();$process.WaitForExit();throw 'Guarded application mode failed to return within the bounded test timeout' }
  $process.Refresh()
  $process.WaitForExit()
  $status=$process.ExitCode
  if($null -eq $status){throw 'Guarded application process completed without a captured exit status'}
  return @{Code=$status;Output=[IO.File]::ReadAllText($out);Error=[IO.File]::ReadAllText($err)}
 } finally { $process.Dispose() }
}
function Refused([string]$State,[string]$Logs,[string]$Label,[string]$Mode='init',[string]$Expected=$refusal) {
 $result=Run-GuardedMode $Mode $State $Logs
 $text=($result.Output+$result.Error).Trim()
 if($result.Code -ne 1 -or $text -ne $Expected) {
  Write-Host ('Guard diagnostic: exit_expected='+[bool]($result.Code -eq 1)+' text_expected='+[bool]($text -eq $Expected)+' stdout_characters='+$result.Output.Length+' stderr_characters='+$result.Error.Length)
 }
 Assert ($result.Code -eq 1 -and $text -eq $Expected) $Label
}

try {
 Assert ([InitElevationProbe]::Elevated()) 'elevated initialization controls use an actually elevated token'
 New-Item -ItemType Directory -Path $root | Out-Null
 $absent=Join-Path $root 'must remain absent'
 Refused (Join-Path $absent 'state') (Join-Path $absent 'logs') 'elevated init refuses with fixed text before creating missing state or diagnostic directories'
 Assert (!(Test-Path -LiteralPath $absent)) 'early elevated refusal creates no selected directory or ancestor'

 $existing=Join-Path $root 'existing state'
 $existingLogs=Join-Path $root 'existing logs'
 foreach($dir in @($existing,$existingLogs)){New-Item -ItemType Directory -Path $dir | Out-Null}
 [IO.File]::WriteAllText((Join-Path $existing 'state-format.txt'),"immichclone-schema-1`n")
 [IO.File]::WriteAllText((Join-Path $existing 'catalog.db'),'synthetic catalog sentinel')
 [IO.File]::WriteAllText((Join-Path $existing 'access-code.txt'),(('a'*64)+"`n"))
 [IO.File]::WriteAllText((Join-Path $existingLogs 'commands.jsonl'),'synthetic command-log sentinel')
 [IO.File]::WriteAllText((Join-Path $existingLogs 'commands.jsonl.previous'),'synthetic prior-log sentinel')
 $beforeState=Snapshot $existing;$beforeLogs=Snapshot $existingLogs
 Refused $existing $existingLogs 'elevated init refuses before opening existing catalog access code marker or command log'
 Assert ((Snapshot $existing) -eq $beforeState -and (Snapshot $existingLogs) -eq $beforeLogs) 'direct refusal preserves every selected state and log descriptor and byte'

 $stateLink=Join-Path $root 'state junction'
 $logLink=Join-Path $root 'log junction'
 New-Item -ItemType Junction -Path $stateLink -Target $existing | Out-Null;$junctions+=$stateLink
 New-Item -ItemType Junction -Path $logLink -Target $existingLogs | Out-Null;$junctions+=$logLink
 Refused $stateLink $logLink 'elevated init refuses actual directory-junction state and logging selectors before writes'
 Assert ((Snapshot $existing) -eq $beforeState -and (Snapshot $existingLogs) -eq $beforeLogs) 'junction selectors preserve outside state and log descriptors and bytes'

 $hardState=Join-Path $root 'hardlinked state'
 $hardLogs=Join-Path $root 'hardlinked logs'
 foreach($dir in @($hardState,$hardLogs)){New-Item -ItemType Directory -Path $dir | Out-Null}
 foreach($name in @('state-format.txt','catalog.db','access-code.txt')){New-Item -ItemType HardLink -Path (Join-Path $hardState $name) -Target (Join-Path $existing $name) | Out-Null}
 $hardLog=Join-Path $hardLogs 'commands.jsonl'
 $outsideLog=Join-Path $existingLogs 'commands.jsonl'
 New-Item -ItemType HardLink -Path $hardLog -Target $outsideLog | Out-Null
 # Real positive control proves these are aliases, not independent copied files.
 [IO.File]::WriteAllText($hardLog,'hardlink positive control')
 Assert ([IO.File]::ReadAllText($outsideLog) -eq 'hardlink positive control') 'actual hard-link fixture redirects a positive-control write to the outside target'
 [IO.File]::WriteAllText($outsideLog,'synthetic command-log sentinel')
 Assert ((Snapshot $existing) -eq $beforeState -and (Snapshot $existingLogs) -eq $beforeLogs) 'hard-link positive control restored all outside descriptors and bytes before candidate refusal'
 $hardStateBefore=Snapshot $hardState;$hardLogsBefore=Snapshot $hardLogs
 Refused $hardState $hardLogs 'elevated init refuses before writing actual hard-linked state and command-log leaves'
 Assert ((Snapshot $hardState) -eq $hardStateBefore -and (Snapshot $hardLogs) -eq $hardLogsBefore -and (Snapshot $existing) -eq $beforeState -and (Snapshot $existingLogs) -eq $beforeLogs) 'hard-linked leaf refusal preserves all selected and outside descriptors and bytes'

 # The command guards must reject before diagnostics/state access, including real
 # redirected selectors; a version or post-open failure is insufficient evidence.
 $guards=@(
  @{Mode='serve';Text='Elevated interactive serving is refused; use the installed LocalService service.'},
  @{Mode='service';Text='Service mode requires the installed LocalService service and Service Control Manager.'},
  @{Mode='unknown-test-mode';Text='Unknown application mode refused before opening files.'}
 )
 foreach($guard in $guards) {
  $modeAbsent=Join-Path $root ('must remain absent-'+$guard.Mode)
  Refused -State (Join-Path $modeAbsent 'state') -Logs (Join-Path $modeAbsent 'logs') -Mode $guard.Mode -Expected $guard.Text -Label 'unsafe elevated or unknown mode refuses before creating directories with exact guard output'
  Assert (!(Test-Path -LiteralPath $modeAbsent)) 'rejected application mode creates no state/log directory or ancestor'
  foreach($selectors in @(@{State=$existing;Logs=$existingLogs},@{State=$stateLink;Logs=$logLink},@{State=$hardState;Logs=$hardLogs})) {
   Refused -State $selectors.State -Logs $selectors.Logs -Mode $guard.Mode -Expected $guard.Text -Label 'unsafe elevated or unknown mode refuses direct junction and hard-linked selectors with exact guard output'
   Assert ((Snapshot $existing) -eq $beforeState -and (Snapshot $existingLogs) -eq $beforeLogs -and (Snapshot $hardState) -eq $hardStateBefore -and (Snapshot $hardLogs) -eq $hardLogsBefore) 'guard refusal preserves every selected and outside state/log descriptor and byte'
  }
 }

 # Show-code is an existing-file read, even with admin authority. The catalog is
 # deliberately not a database: success proves Store.Open was never called.
 $show=Join-Path $root 'show-code state'
 New-Item -ItemType Directory -Path $show | Out-Null
 $showCode=Join-Path $show 'access-code.txt'
 [IO.File]::WriteAllText($showCode,(('c'*64)+"`r`n"))
 [IO.File]::WriteAllText((Join-Path $show 'catalog.db'),'non-database catalog sentinel; must never be opened')
 [IO.File]::WriteAllText((Join-Path $show 'state-format.txt'),'unchanged marker sentinel')
 $showBefore=Snapshot $show
 $showLogs=Join-Path $root 'show-code logs must remain absent'
 $shown=Run-GuardedMode 'show-code' $show $showLogs
 Assert ($shown.Code -eq 0 -and $shown.Output.Trim() -eq ('c'*64) -and $shown.Error.Trim() -eq '') 'elevated show-code reads the existing valid code without opening a non-database catalog'
 Assert ((Snapshot $show) -eq $showBefore -and !(Test-Path -LiteralPath $showLogs)) 'show-code success preserves state bytes/descriptors and creates no diagnostic files'

 $showRefusal='Existing access code unavailable; no state was initialized.'
 $missingShow=Join-Path $root 'missing-code state'
 New-Item -ItemType Directory -Path $missingShow | Out-Null
 [IO.File]::WriteAllText((Join-Path $missingShow 'catalog.db'),'missing-code catalog sentinel')
 $missingBefore=Snapshot $missingShow
 Refused -State $missingShow -Logs $showLogs -Mode 'show-code' -Expected $showRefusal -Label 'missing access code is refused without generating a replacement'
 Assert ((Snapshot $missingShow) -eq $missingBefore -and !(Test-Path -LiteralPath (Join-Path $missingShow 'access-code.txt')) -and !(Test-Path -LiteralPath $showLogs)) 'missing-code refusal creates no code marker storage directories or logs'
 $showJunction=Join-Path $root 'show-code junction'
 New-Item -ItemType Junction -Path $showJunction -Target $show | Out-Null;$junctions+=$showJunction
 Refused -State $showJunction -Logs $showLogs -Mode 'show-code' -Expected $showRefusal -Label 'show-code refuses an actual junction root before reading the outside code'
 Assert ((Snapshot $show) -eq $showBefore -and !(Test-Path -LiteralPath $showLogs)) 'show-code junction refusal preserves outside code/catalog/marker and no log output'
 $showHard=Join-Path $root 'show-code hardlink state'
 New-Item -ItemType Directory -Path $showHard | Out-Null
 $showAlias=Join-Path $showHard 'access-code.txt'
 New-Item -ItemType HardLink -Path $showAlias -Target $showCode | Out-Null
 [IO.File]::WriteAllText($showAlias,'synthetic hardlink code positive control')
 Assert ([IO.File]::ReadAllText($showCode) -eq 'synthetic hardlink code positive control') 'show-code hardlink fixture really aliases the selected outside code'
 [IO.File]::WriteAllText($showCode,(('c'*64)+"`r`n"))
 $showHardBefore=Snapshot $showHard
 Refused -State $showHard -Logs $showLogs -Mode 'show-code' -Expected $showRefusal -Label 'show-code refuses an actual hard-linked code leaf instead of opening it'
 Assert ((Snapshot $showHard) -eq $showHardBefore -and (Snapshot $show) -eq $showBefore -and !(Test-Path -LiteralPath $showLogs)) 'show-code hardlink refusal preserves selected/outside bytes and descriptors without diagnostics'
} finally {
 foreach($path in $junctions){if(Test-Path -LiteralPath $path){[IO.Directory]::Delete($path)}}
 Remove-Item -LiteralPath $root -Recurse -Force -ErrorAction SilentlyContinue
}

exit 0
