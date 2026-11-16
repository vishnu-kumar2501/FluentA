; Inno Setup 6 Script for FluentA
; Per-user installation (No administrator / UAC elevation required)
; Installs to %LOCALAPPDATA%\Programs\FluentA

#define MyAppName "FluentA"
#define MyAppVersion "1.0.0"
#define MyAppPublisher "FluentA"
#define MyAppURL "https://github.com/FluentA"
#define MyAppExeName "FluentA.exe"
#define MyAppIcon "..\desktop\fluenta.ico"

[Setup]
; Unique GUID for FluentA
AppId={{8B19C9F0-55A2-4D93-94D9-052C88F4A1E2}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}

; Per-user install: no Administrator elevation required
DefaultDirName={localappdata}\Programs\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog

; Output configuration
OutputDir=..\dist
OutputBaseFilename=FluentASetup
SetupIconFile={#MyAppIcon}
UninstallDisplayIcon={app}\fluenta.ico
UninstallDisplayName={#MyAppName}

; Compression
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern

; Process management: cleanly close running instance before install / upgrade
CloseApplications=force
CloseApplicationsFilter={#MyAppExeName}
ArchitecturesInstallIn64BitMode=x64compatible

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "startupicon"; Description: "Start FluentA automatically when Windows starts"; GroupDescription: "Startup options:"; Flags: unchecked

[Files]
Source: "..\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#MyAppIcon}"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\PRIVACY.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{userprograms}\{#MyAppName}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\fluenta.ico"
Name: "{userprograms}\{#MyAppName}\FluentA Privacy Policy"; Filename: "{app}\PRIVACY.md"
Name: "{userdesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\fluenta.ico"; Tasks: desktopicon

[Registry]
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "{#MyAppName}"; ValueData: """{app}\{#MyAppExeName}"""; Flags: uninsdeletevalue; Tasks: startupicon

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
Type: filesandordirs; Name: "{localappdata}\FluentA"
Type: filesandordirs; Name: "{app}"

[Code]
const
  WV2_URL = 'https://go.microsoft.com/fwlink/p/?LinkId=2124703';

// Clean up all running processes, files, directories, and registry keys on uninstall
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  ResultCode: Integer;
begin
  if CurUninstallStep = usUninstall then
  begin
    // Terminate any running FluentA background or tray process before file deletion
    Exec('taskkill.exe', '/f /im {#MyAppExeName}', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
    Sleep(250);
  end
  else if CurUninstallStep = usPostUninstall then
  begin
    // Completely remove runtime settings, history, logs, and WebView2 cache
    DelTree(ExpandConstant('{localappdata}\FluentA'), True, True, True);
    // Remove the installation directory if anything remains
    DelTree(ExpandConstant('{app}'), True, True, True);
    // Remove autostart registry key if set
    RegDeleteValue(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Run', '{#MyAppName}');
  end;
end;

function CheckWebView2RegKey(RootKey: Integer; SubKey: String): Boolean;
var
  VersionStr: String;
begin
  Result := False;
  if RegQueryStringValue(RootKey, SubKey, 'pv', VersionStr) then
  begin
    if (Length(VersionStr) > 0) and (VersionStr <> '0.0.0.0') then
      Result := True;
  end;
end;

// Checks if Microsoft Edge WebView2 Runtime is installed
function IsWebView2Installed(): Boolean;
begin
  Result := False;

  // 1. Primary Windows 10/11 GUID {F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}
  if CheckWebView2RegKey(HKLM32, 'SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}') then
    Result := True
  else if CheckWebView2RegKey(HKLM64, 'SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}') then
    Result := True
  else if CheckWebView2RegKey(HKLM, 'SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}') then
    Result := True
  else if CheckWebView2RegKey(HKCU, 'Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}') then
    Result := True

  // 2. Legacy / Alternative GUID {F3017226-6F4A-4765-AE4E-64A2DE02E181}
  else if CheckWebView2RegKey(HKLM32, 'SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-6F4A-4765-AE4E-64A2DE02E181}') then
    Result := True
  else if CheckWebView2RegKey(HKLM64, 'SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-6F4A-4765-AE4E-64A2DE02E181}') then
    Result := True
  else if CheckWebView2RegKey(HKLM, 'SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-6F4A-4765-AE4E-64A2DE02E181}') then
    Result := True
  else if CheckWebView2RegKey(HKCU, 'Software\Microsoft\EdgeUpdate\Clients\{F3017226-6F4A-4765-AE4E-64A2DE02E181}') then
    Result := True

  // 3. Fallback: check filesystem installation directories
  else if DirExists(ExpandConstant('{commonpf32}\Microsoft\EdgeWebView\Application')) then
    Result := True
  else if DirExists(ExpandConstant('{commonpf64}\Microsoft\EdgeWebView\Application')) then
    Result := True
  else if DirExists('C:\Program Files (x86)\Microsoft\EdgeWebView\Application') then
    Result := True
  else if DirExists('C:\Program Files\Microsoft\EdgeWebView\Application') then
    Result := True;
end;

// Prompts to download and install WebView2 Evergreen Bootstrapper if missing
function InitializeSetup(): Boolean;
var
  ResultCode: Integer;
  Msg: String;
  PowerShellCmd: String;
begin
  Result := True;

  if not IsWebView2Installed() then
  begin
    Msg := 'FluentA requires the Microsoft Edge WebView2 Runtime to display its user interface.' + #13#10 + #13#10 +
           'It appears to be missing on your system.' + #13#10#13#10 +
           'Would you like the installer to automatically download and install it now?';

    if MsgBox(Msg, mbConfirmation, MB_YESNO) = IDYES then
    begin
      PowerShellCmd :=
        '-NoProfile -ExecutionPolicy Bypass -Command ' +
        '"[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; ' +
        '$dl = Join-Path $env:TEMP ''MicrosoftEdgeWebview2Setup.exe''; ' +
        '(New-Object System.Net.WebClient).DownloadFile(''' + WV2_URL + ''', $dl); ' +
        'Start-Process -FilePath $dl -ArgumentList ''/silent'',''/install'' -Wait"';

      Exec('powershell.exe', PowerShellCmd, '', SW_SHOW, ewWaitUntilTerminated, ResultCode);

      if not IsWebView2Installed() then
      begin
        MsgBox(
          'WebView2 installation could not be verified.' + #13#10 +
          'FluentA may not display properly until WebView2 is installed.' + #13#10#13#10 +
          'You can download it manually from:' + #13#10 +
          'https://developer.microsoft.com/microsoft-edge/webview2/',
          mbInformation, MB_OK
        );
      end;
    end;
  end;
end;

