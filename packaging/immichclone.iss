#ifndef AppVersion
#define AppVersion "0.1.13"
#endif
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
Source: "..\dist\immichclone.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "Export-Diagnostics.ps1"; DestDir: "{app}"; Flags: ignoreversion
Source: "Manage-Service.ps1"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\THIRD-PARTY-LICENSES.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\docs\prototype-notes.txt"; DestDir: "{app}"; Flags: ignoreversion

Source: "..\dist\media\*"; DestDir: "{app}\media"; Flags: ignoreversion recursesubdirs createallsubdirs

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
Type: filesandordirs; Name: "{app}\media"
Type: files; Name: "{app}\rollback\immichclone.exe"
Type: files; Name: "{app}\rollback\catalog.db"
Type: files; Name: "{app}\rollback\access-code.txt"
Type: dirifempty; Name: "{app}\rollback"

[Code]
var StatePage: TInputDirWizardPage; ManagementFailed: Boolean;

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

function RunManagement(Action: String; var ResultCode: Integer): Boolean;
begin
  Result := Exec(ExpandConstant('{sys}\WindowsPowerShell\v1.0\powershell.exe'),
    '-NoProfile -NonInteractive -ExecutionPolicy Bypass -File "' + ExpandConstant('{app}\Manage-Service.ps1') +
    '" -Action ' + Action + ' -AppDir "' + ExpandConstant('{app}') + '" -DataDir "' + GetState('') + '"',
    '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
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
  if FileExists(ExpandConstant('{app}\Manage-Service.ps1')) then
    if not RunManagement('Prepare', ResultCode) or (ResultCode <> 0) then
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
