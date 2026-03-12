; Полноценный установщик: один exe, ставит приложение + WebView2 (тихо). Распространяйте только этот файл.
Unicode true
!include "MUI2.nsh"
!include "FileFunc.nsh"

; Имена и версия (совпадают с wails.json)
!define PRODUCT_NAME "FCS AutoReport"
!define COMPANY_NAME "FCS"
!define VERSION "1.0.0"
!define EXE_NAME "FCS AutoReport.exe"
!define UNINSTALL_KEY "FCS_FCS_AutoReport"

; Путь к exe задаётся при сборке: makensis -DAPP_EXE=..\..\bin\FCS AutoReport.exe
!ifndef APP_EXE
  !define APP_EXE "..\..\bin\FCS AutoReport.exe"
!endif

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN "$INSTDIR\${EXE_NAME}"
!define MUI_FINISHPAGE_RUN_TEXT "Запустить FCS AutoReport"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\..\LICENSE.txt"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Name "${PRODUCT_NAME}"
OutFile "..\..\bin\FCS_AutoReport_Setup.exe"
InstallDir "$PROGRAMFILES64\${COMPANY_NAME}\${PRODUCT_NAME}"
InstallDirRegKey HKCU "Software\${UNINSTALL_KEY}" "InstallPath"
RequestExecutionLevel admin

VIProductVersion "${VERSION}.0"
VIFileVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "${PRODUCT_NAME}"
VIAddVersionKey "CompanyName" "${COMPANY_NAME}"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "LegalCopyright" "FCS"

; Тихая установка WebView2, если ещё не установлен
Function InstallWebView2Silent
  ReadRegStr $0 HKLM "SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
  ${If} $0 == ""
    SetDetailsPrint both
    DetailPrint "Установка WebView2 (один раз)..."
    SetDetailsPrint listonly
    SetOutPath $INSTDIR
    File "tmp\MicrosoftEdgeWebview2Setup.exe"
    ExecWait '"$INSTDIR\MicrosoftEdgeWebview2Setup.exe" /silent /install' $1
    Delete "$INSTDIR\MicrosoftEdgeWebview2Setup.exe"
    SetDetailsPrint both
  ${EndIf}
FunctionEnd

Section "MainSection" SEC01
  SetOutPath $INSTDIR
  ; Копируем exe приложения
  File "${APP_EXE}"
  Call InstallWebView2Silent
  CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}.lnk" "$INSTDIR\${EXE_NAME}"
  CreateShortCut "$DESKTOP\${PRODUCT_NAME}.lnk" "$INSTDIR\${EXE_NAME}"
  WriteRegStr HKCU "Software\${UNINSTALL_KEY}" "InstallPath" $INSTDIR
  WriteUninstaller "$INSTDIR\Uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINSTALL_KEY}" "DisplayName" "${PRODUCT_NAME}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINSTALL_KEY}" "UninstallString" "$INSTDIR\Uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINSTALL_KEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINSTALL_KEY}" "Publisher" "${COMPANY_NAME}"
SectionEnd

Section "Uninstall"
  RMDir /r "$AppData\${EXE_NAME}"
  RMDir /r $INSTDIR
  Delete "$SMPROGRAMS\${PRODUCT_NAME}.lnk"
  Delete "$DESKTOP\${PRODUCT_NAME}.lnk"
  DeleteRegKey HKCU "Software\${UNINSTALL_KEY}"
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINSTALL_KEY}"
SectionEnd
