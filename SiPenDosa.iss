; ========================================================================
; Inno Setup Script: SiPenDosa Windows Installer
; "Asisten yang rela 'berdosa' demi mengingatkan dosen agar mahasiswa tidak sungkan"
; ========================================================================

#define MyAppName "SiPenDosa"
#define MyAppVersion "1.2.0"
#define MyAppPublisher "SiPenDosa Team"
#define MyAppURL "http://localhost:8473"
#define MyAppExeName "sipen.exe"

[Setup]
; NOTE: The value of AppId uniquely identifies this application.
AppId={{D37E88A1-4F29-4A1B-8DF3-9E3B87A845C2}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}
DefaultDirName={localappdata}\Programs\{#MyAppName}
DisableProgramGroupPage=yes
; PrivilegesRequired: lowest means no admin elevation required (installs for current user)
PrivilegesRequired=lowest
OutputDir=.
OutputBaseFilename=SiPenDosa-Wizard-Setup
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern

[Languages]
Name: "indonesian"; MessagesFile: "compiler:Languages\Indonesian.isl"
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "sipen.exe"; DestDir: "{app}"; Flags: ignoreversion
; NOTE: Don't use "Flags: ignoreversion" on any shared system files

[Dirs]
Name: "{app}\data"; Permissions: users-full
Name: "{app}\session"; Permissions: users-full

[Icons]
Name: "{autoprograms}\{#MyAppName}\{#MyAppName} (Terminal)"; Filename: "{app}\{#MyAppExeName}"; WorkingDir: "{app}"
Name: "{autoprograms}\{#MyAppName}\Buka Web Dashboard"; Filename: "{#MyAppURL}"
Name: "{autoprograms}\{#MyAppName}\{cm:UninstallProgram,{#MyAppName}}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; WorkingDir: "{app}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent
