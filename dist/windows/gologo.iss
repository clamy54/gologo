; Installeur Windows pour GoLogo (Inno Setup 6).
; Prerequis : avoir compile l'executable avec tools/build/build-windows.ps1
; (l'exe attendu est tools/build/bin/windows/gologo.exe).
; Compilation de l'installeur : ISCC.exe gologo.iss  (ou via l'IDE Inno Setup).

#define AppName "GoLogo"
; La version peut etre imposee a la compilation : ISCC.exe /DAppVersion=x.y gologo.iss
#ifndef AppVersion
  #define AppVersion "2.2"
#endif
#define AppPublisher "Cyril Lamy"
#define AppExe "gologo.exe"

[Setup]
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
DefaultDirName={autopf}\GoLogo
DefaultGroupName=GoLogo
DisableProgramGroupPage=yes
UninstallDisplayIcon={app}\{#AppExe}
LicenseFile=..\..\LICENSE
OutputDir=output
OutputBaseFilename=GoLogo-Setup-{#AppVersion}
SetupIconFile=..\..\tools\build\icons\gologo.ico
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
; L'executable est un binaire amd64 : on n'installe que la ou il peut tourner
; (x64, ou Arm64 avec emulation x64), pas sur un Windows 32 bits.
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible

[Languages]
Name: "fr"; MessagesFile: "compiler:Languages\French.isl"
Name: "en"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"

[Files]
Source: "..\..\tools\build\bin\windows\gologo.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\src\examples\*"; DestDir: "{app}\examples"; Flags: recursesubdirs createallsubdirs
Source: "..\..\tools\build\icons\gologo.ico"; DestDir: "{app}"
Source: "..\..\LICENSE"; DestDir: "{app}"; DestName: "LICENSE.txt"

[Icons]
Name: "{group}\GoLogo"; Filename: "{app}\{#AppExe}"; IconFilename: "{app}\gologo.ico"
Name: "{group}\{cm:UninstallProgram,GoLogo}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\GoLogo"; Filename: "{app}\{#AppExe}"; IconFilename: "{app}\gologo.ico"; Tasks: desktopicon

[Run]
Filename: "{app}\{#AppExe}"; Description: "{cm:LaunchProgram,GoLogo}"; Flags: nowait postinstall skipifsilent
