!define APP_NAME "Katenary"
!define COMPANY_NAME "Katenary"

OutFile "katenary_installer.exe"
InstallDir "$LOCALAPPDATA\Katenary"
RequestExecutionLevel user

!include "MUI2.nsh"
!addplugindir "."

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "French"

Name "${APP_NAME} ${APP_VERSION}"

Section "Install"
  SetOutPath "$INSTDIR"
  File "..\dist\katenary.exe"
  WriteUninstaller "$INSTDIR\uninstall.exe"

  ; Manipulation PATH utilisateur
  EnVar::SetHKCU
  Pop $0

  EnVar::AddValue "Path" "$INSTDIR"
  Pop $0 ; 0 = succès

SectionEnd

Section "Uninstall"
  EnVar::SetHKCU
  Pop $0

  EnVar::DeleteValue "Path" "$INSTDIR"
  Pop $0

  Delete "$INSTDIR\katenary.exe"
  Delete "$INSTDIR\uninstall.exe"
SectionEnd
