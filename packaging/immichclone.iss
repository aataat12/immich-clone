#ifndef AppVersion
#define AppVersion "0.1.23"
#endif
#define ManagementSHA256 GetSHA256OfFile(AddBackslash(SourcePath) + "Manage-Service.ps1")
#define BoundaryHelperSHA256 GetSHA256OfFile(AddBackslash(SourcePath) + "..\dist\installer-files.exe")
#define StorageSecuritySHA256 GetSHA256OfFile(AddBackslash(SourcePath) + "Storage-Security.ps1")
[Setup]
AppId={{5DF7680B-6F42-4BAF-AD97-5A3095728DFA}
AppName=Immich Clone Prototype
AppVersion={#AppVersion}
DefaultDirName={autopf}\ImmichClone
DefaultGroupName=Immich Clone Prototype
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0
PrivilegesRequired=admin
DisableProgramGroupPage=yes
OutputDir=..\dist
OutputBaseFilename=ImmichClone-{#AppVersion}-prototype-x64-setup
Compression=lzma2
SolidCompression=yes
CloseApplications=no
SetupLogging=no
UninstallDisplayIcon={app}\immichclone.exe
InfoBeforeFile=..\docs\prototype-notes.txt

[Files]
; Bootstrap only; normal Inno copying never writes application payload files.
Source: "Manage-Service.ps1"; Flags: dontcopy
Source: "Storage-Security.ps1"; Flags: dontcopy
Source: "..\dist\installer-files.exe"; Flags: dontcopy
Source: "Manage-Service.ps1"; DestDir: "{tmp}\payload"; Flags: dontcopy
Source: "Storage-Security.ps1"; DestDir: "{tmp}\payload"; Flags: dontcopy
Source: "..\dist\installer-files.exe"; DestDir: "{tmp}\payload"; Flags: dontcopy
Source: "..\dist\immichclone.exe"; DestDir: "{tmp}\payload"; Flags: dontcopy
Source: "Export-Diagnostics.ps1"; DestDir: "{tmp}\payload"; Flags: dontcopy
Source: "..\dist\THIRD-PARTY-LICENSES.txt"; DestDir: "{tmp}\payload"; Flags: dontcopy
Source: "..\docs\prototype-notes.txt"; DestDir: "{tmp}\payload"; Flags: dontcopy
Source: "..\dist\media\*"; DestDir: "{tmp}\payload\media"; Flags: dontcopy recursesubdirs createallsubdirs
Source: "..\dist\ai\*"; DestDir: "{tmp}\payload\ai"; Flags: dontcopy recursesubdirs createallsubdirs

[Registry]
Root: HKLM; Subkey: "Software\ImmichClone"; ValueType: string; ValueName: "State"; ValueData: "{code:GetState}"; Flags: uninsdeletekey

[Icons]
Name: "{group}\Open library"; Filename: "http://127.0.0.1:2283"
Name: "{group}\View access code"; Filename: "{sys}\notepad.exe"; Parameters: """{code:GetState}\access-code.txt"""
Name: "{group}\Prototype limitations"; Filename: "{app}\prototype-notes.txt"

Name: "{group}\Export diagnostic log"; Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\Export-Diagnostics.ps1"""

[UninstallRun]
Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -NonInteractive -ExecutionPolicy Bypass -File ""{app}\Manage-Service.ps1"" -Action Uninstall -AppDir ""{app}"" -DataDir ""{code:GetState}"""; RunOnceId: "RemoveImmichService"; Flags: runhidden waituntilterminated

[UninstallDelete]
; Native-published fixed files retain removal behavior; no app-root wildcard.
Type: files; Name: "{app}\Manage-Service.ps1"
Type: files; Name: "{app}\Storage-Security.ps1"
Type: files; Name: "{app}\installer-files.exe"
Type: files; Name: "{app}\immichclone.exe"
Type: files; Name: "{app}\Export-Diagnostics.ps1"
Type: files; Name: "{app}\THIRD-PARTY-LICENSES.txt"
Type: files; Name: "{app}\prototype-notes.txt"
Type: filesandordirs; Name: "{app}\media"
Type: filesandordirs; Name: "{app}\ai"
Type: files; Name: "{app}\rollback\immichclone.exe"
Type: files; Name: "{app}\rollback\catalog.db"
Type: files; Name: "{app}\rollback\access-code.txt"
Type: files; Name: "{app}\rollback\snapshot-ready.json"
Type: dirifempty; Name: "{app}\rollback"

[Code]
var StatePage: TInputDirWizardPage; ManagementFailed: Boolean;
    ManagementHandle, SecurityHandle, BoundaryHandle: Integer;

function OpenReadLockedFile(Name: String; Access, Sharing: LongWord;
  Security: Integer; Creation, Attributes: LongWord; Template: Integer): Integer;
  external 'CreateFileW@kernel32.dll stdcall';
function CloseLockedFile(Handle: Integer): Boolean;
  external 'CloseHandle@kernel32.dll stdcall';

procedure ClosePreparationHandles;
begin
  if (ManagementHandle <> 0) and (ManagementHandle <> -1) then
    CloseLockedFile(ManagementHandle);
  if (SecurityHandle <> 0) and (SecurityHandle <> -1) then
    CloseLockedFile(SecurityHandle);
  if (BoundaryHandle <> 0) and (BoundaryHandle <> -1) then
    CloseLockedFile(BoundaryHandle);
  ManagementHandle := 0;
  SecurityHandle := 0;
  BoundaryHandle := 0;
end;

procedure DeinitializeSetup;
begin
  ClosePreparationHandles;
end;

function GetState(Param: String): String;
begin
  if StatePage <> nil then Result := StatePage.Values[0]
  else if not RegQueryStringValue(HKLM, 'Software\ImmichClone', 'State', Result) then
    Result := ExpandConstant('{commonappdata}\ImmichClone');
end;

procedure InitializeWizard;
var Previous: String;
begin
  StatePage := CreateInputDirPage(wpSelectDir, 'Application data',
    'Choose a folder outside cloud sync folders.',
    'This folder stores managed copies of uploaded originals, catalog and previews. Uninstall preserves it. This is not your external source folder.', False, '');
  StatePage.Add('Application data folder:');
  if not RegQueryStringValue(HKLM, 'Software\ImmichClone', 'State', Previous) then
    Previous := ExpandConstant('{commonappdata}\ImmichClone');
  StatePage.Values[0] := ExpandConstant('{param:STATE|' + Previous + '}');
end;

function NextButtonClick(CurPageID: Integer): Boolean;
var Previous: String;
begin
  Result := True;
  if CurPageID = StatePage.ID then begin
    if RegQueryStringValue(HKLM, 'Software\ImmichClone', 'State', Previous) and
       (CompareText(RemoveBackslashUnlessRoot(Previous), RemoveBackslashUnlessRoot(StatePage.Values[0])) <> 0) then begin
      MsgBox('Changing an existing data location requires a verified backup and restore, not an upgrade.', mbError, MB_OK);
      Result := False;
    end;
  end;
end;

function RunManagementFrom(ScriptPath, Action: String; var ResultCode: Integer): Boolean;
begin
  Result := Exec(ExpandConstant('{sys}\WindowsPowerShell\v1.0\powershell.exe'),
    '-NoProfile -NonInteractive -ExecutionPolicy Bypass -File "' + ScriptPath +
    '" -Action ' + Action + ' -AppDir "' + ExpandConstant('{app}') + '" -DataDir "' + GetState('') + '" -BoundaryHelper "' + ExpandConstant('{tmp}\installer-files.exe') + '" -PayloadSource "' + ExpandConstant('{tmp}\payload') + '" -PayloadBackup "' + ExpandConstant('{tmp}\payload-backup') + '"',
    '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
end;

function RunManagement(Action: String; var ResultCode: Integer): Boolean;
begin
  { Use the new payload for both Prepare and Install, never retained app code. }
  Result := RunManagementFrom(ExpandConstant('{tmp}\Manage-Service.ps1'), Action, ResultCode);
end;

function RunBundledPreparation(var ResultCode: Integer): Boolean;
begin
  Result := False;
  ResultCode := -1;
  try
    ClosePreparationHandles;
    ExtractTemporaryFile('Manage-Service.ps1');
    ExtractTemporaryFile('Storage-Security.ps1');
    ExtractTemporaryFile('installer-files.exe');
    { Hold read-only handles without WRITE/DELETE sharing from before hashing
      until Setup exits. A writer already holding access also makes this fail.
      These handles cover the exact staged files, not all filesystem races. }
    ManagementHandle := OpenReadLockedFile(ExpandConstant('{tmp}\Manage-Service.ps1'),
      $80000000, 1, 0, 3, $00200000, 0);
    SecurityHandle := OpenReadLockedFile(ExpandConstant('{tmp}\Storage-Security.ps1'),
      $80000000, 1, 0, 3, $00200000, 0);
    BoundaryHandle := OpenReadLockedFile(ExpandConstant('{tmp}\installer-files.exe'),
      $80000000, 1, 0, 3, $00200000, 0);
    if (ManagementHandle = -1) or (SecurityHandle = -1) or (BoundaryHandle = -1) then exit;
    if (CompareText(GetSHA256OfFile(ExpandConstant('{tmp}\Manage-Service.ps1')),
         '{#ManagementSHA256}') <> 0) or
       (CompareText(GetSHA256OfFile(ExpandConstant('{tmp}\Storage-Security.ps1')),
         '{#StorageSecuritySHA256}') <> 0) or
       (CompareText(GetSHA256OfFile(ExpandConstant('{tmp}\installer-files.exe')),
         '{#BoundaryHelperSHA256}') <> 0) then
      exit;
    ExtractTemporaryFiles('{tmp}\payload\*');
    { The script dot-sources its sibling helper from this extracted payload.
      Never execute the retained installed preparer before new validation. }
    Result := RunManagementFrom(ExpandConstant('{tmp}\Manage-Service.ps1'), 'Prepare', ResultCode);
  except
    { Extraction/hash failures abort before service actions or file publication. }
    Result := False;
    ResultCode := -1;
  end;
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var ResultCode: Integer; Previous: String;
begin
  Result := '';
  if RegQueryStringValue(HKLM, 'Software\ImmichClone', 'State', Previous) and
     (CompareText(RemoveBackslashUnlessRoot(Previous), RemoveBackslashUnlessRoot(GetState(''))) <> 0) then begin
    Result := 'Changing the data location requires backup and restore. Upgrade refused.';
    exit;
  end;
  if not RunBundledPreparation(ResultCode) or (ResultCode <> 0) then
    Result := 'Could not prepare the existing service safely. No new files were installed.';
end;

procedure CurStepChanged(CurStep: TSetupStep);
var ResultCode: Integer;
begin
  if CurStep = ssPostInstall then
    if not RunManagement('Install', ResultCode) or (ResultCode <> 0) then begin
      ManagementFailed := True;
      RaiseException('Service installation failed. Previous state was retained; review the service before retrying.');
    end;
end;

function GetCustomSetupExitCode: Integer;
begin
  if ManagementFailed then Result := 1 else Result := 0;
end;
