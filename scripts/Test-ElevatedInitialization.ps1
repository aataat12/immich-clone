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
function Refused([string]$State,[string]$Logs,[string]$Label) {
 $script:attempt++
 $out=Join-Path $root ('output-'+$script:attempt+'.txt')
 $err=Join-Path $root ('error-'+$script:attempt+'.txt')
 $process=Start-Process -FilePath $Binary -ArgumentList @('-mode','init','-data',('"'+$State+'"'),'-logs',('"'+$Logs+'"')) -PassThru -Wait -RedirectStandardOutput $out -RedirectStandardError $err
 $text=([IO.File]::ReadAllText($out)+[IO.File]::ReadAllText($err)).Trim()
 Assert ($process.ExitCode -eq 1 -and $text -eq $refusal) $Label
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
} finally {
 foreach($path in $junctions){if(Test-Path -LiteralPath $path){[IO.Directory]::Delete($path)}}
 Remove-Item -LiteralPath $root -Recurse -Force -ErrorAction SilentlyContinue
}

exit 0
