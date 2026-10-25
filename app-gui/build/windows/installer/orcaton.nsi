
Unicode true
ManifestDPIAware true
ManifestSupportedOS all
RequestExecutionLevel admin
SetCompressor /SOLID lzma

!ifndef VERSION
  !error "VERSION is not set (/DVERSION=0.0.46)"
!endif
!ifndef ARCH
  !error "ARCH is not set (/DARCH=x64 or /DARCH=arm64)"
!endif
!ifndef SRC
  !error "SRC is not set (/DSRC=<folder with orcaton-gui.exe>)"
!endif
!ifndef OUTFILE
  !define OUTFILE "Orcaton-${VERSION}-windows-${ARCH}-setup.exe"
!endif
!if "${ARCH}" != "x64"
!if "${ARCH}" != "arm64"
  !error "ARCH must be x64 or arm64"
!endif
!endif

!define APP "Orcaton"
!define PUBLISHER "Athlon1323"
!define UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\Orcaton"
!define EXE "orcaton-gui.exe"
!define WEBVIEW2_ID "{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}"
!define WEBVIEW2_URL "https://go.microsoft.com/fwlink/p/?LinkId=2124703"

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "Sections.nsh"
!include "FileFunc.nsh"
!include "WinVer.nsh"
!include "WinMessages.nsh"
!include "x64.nsh"

Name "${APP}"
Caption "${APP} ${VERSION} Setup"
UninstallCaption "${APP} ${VERSION} Uninstall"
BrandingText "${APP} ${VERSION} (${ARCH})"
OutFile "${OUTFILE}"
InstallDir "$PROGRAMFILES64\Orcaton"
ShowInstDetails show
ShowUninstDetails show

VIProductVersion "${VERSION}.0"
VIAddVersionKey /LANG=1033 "ProductName" "${APP}"
VIAddVersionKey /LANG=1033 "CompanyName" "${APP}"
VIAddVersionKey /LANG=1033 "FileDescription" "${APP} Setup"
VIAddVersionKey /LANG=1033 "FileVersion" "${VERSION}"
VIAddVersionKey /LANG=1033 "ProductVersion" "${VERSION}"
VIAddVersionKey /LANG=1033 "LegalCopyright" "Copyright 2026 Athlon1323. Apache License 2.0"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_ABORTWARNING
!define MUI_COMPONENTSPAGE_NODESC

!define MUI_WELCOMEPAGE_TEXT "This will install ${APP} ${VERSION} (${ARCH}) for all users of this computer.$\r$\n$\r$\n${APP} synchronizes multicam video and sound by audio (FCP7 XML and DRT timelines).$\r$\n$\r$\nFFmpeg is not included: if it is missing, the ${APP} window offers to install it on the first start.$\r$\n$\r$\nClick Next to continue."
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "${SRC}\license-page.txt"
!insertmacro MUI_PAGE_COMPONENTS
!define MUI_PAGE_CUSTOMFUNCTION_LEAVE DirectoryLeave
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN
!define MUI_FINISHPAGE_RUN_TEXT "Run ${APP}"
!define MUI_FINISHPAGE_RUN_FUNCTION RunApp
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_COMPONENTS
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Var OldDir   ; прежняя установка для пользователя (%LOCALAPPDATA%\Programs\Orcaton)


!macro CHECK_RUNNING_FUNC un
Function ${un}CheckRunning
  StrCpy $R9 0
  retry:
  StrCpy $R8 ""
  ${If} ${FileExists} "$R6\${EXE}"
    ClearErrors
    FileOpen $R7 "$R6\${EXE}" a
    ${If} ${Errors}
      StrCpy $R8 "${APP} window"
    ${Else}
      FileClose $R7
    ${EndIf}
  ${EndIf}
  ${If} $R8 == ""
  ${AndIf} ${FileExists} "$R6\orcaton.exe"
    ClearErrors
    FileOpen $R7 "$R6\orcaton.exe" a
    ${If} ${Errors}
      StrCpy $R8 "orcaton command"
    ${Else}
      FileClose $R7
    ${EndIf}
  ${EndIf}
  ${If} $R8 != ""
    MessageBox MB_RETRYCANCEL|MB_ICONEXCLAMATION "${APP} is running ($R8).$\r$\nClose it and click Retry." /SD IDCANCEL IDRETRY retry
    StrCpy $R9 1
  ${EndIf}
FunctionEnd
!macroend
!insertmacro CHECK_RUNNING_FUNC ""
!insertmacro CHECK_RUNNING_FUNC "un."

!macro RUN_PATH_TOOL action scope dir extra
  InitPluginsDir
  SetOutPath "$PLUGINSDIR"
  File "path.ps1"
  nsExec::ExecToLog '"$SYSDIR\WindowsPowerShell\v1.0\powershell.exe" -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "$PLUGINSDIR\path.ps1" -Action ${action} -Scope ${scope} -Dir "${dir}" ${extra}'
  Pop $R0
  SetOutPath "$TEMP"
  SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 "STR:Environment" /TIMEOUT=3000
!macroend

!macro IS_ADMIN_FUNC un
Function ${un}IsAdmin
  UserInfo::GetAccountType
  Pop $0
  ${If} $0 != "admin"
    MessageBox MB_OK|MB_ICONSTOP "Administrator rights are needed: ${APP} is installed in Program Files for all users." /SD IDOK
    SetErrorLevel 740
    Quit
  ${EndIf}
FunctionEnd
!macroend
!insertmacro IS_ADMIN_FUNC ""
!insertmacro IS_ADMIN_FUNC "un."


Section "${APP} (required)" SecMain
  SectionIn RO
  StrCpy $R6 "$INSTDIR"
  Call CheckRunning
  ${If} $R9 == 1
    Abort "${APP} is running — installation cancelled."
  ${EndIf}
  Call RemoveUserCopy
  SetOutPath "$INSTDIR"
  Delete "$INSTDIR\uninstall.ps1"
  Delete "$INSTDIR\install-manifest.txt"
  File "${SRC}\${EXE}"
  File "${SRC}\orcaton.exe"
  File "${SRC}\LICENSE"
  File "${SRC}\NOTICE"
  File "${SRC}\THIRD-PARTY-NOTICES.md"
  File "${SRC}\VERSION"
  WriteUninstaller "$INSTDIR\uninstall.exe"

  CreateShortcut "$SMPROGRAMS\${APP}.lnk" "$INSTDIR\${EXE}" "" "$INSTDIR\${EXE}" 0 SW_SHOWNORMAL "" "${APP} — multicam audio sync"

  WriteRegStr HKLM "${UNINST_KEY}" "DisplayName" "${APP}"
  WriteRegStr HKLM "${UNINST_KEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "${UNINST_KEY}" "Publisher" "${PUBLISHER}"
  WriteRegStr HKLM "${UNINST_KEY}" "Comments" "Multicam audio sync (FCP7 XML, DRT)"
  WriteRegStr HKLM "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKLM "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\${EXE},0"
  WriteRegStr HKLM "${UNINST_KEY}" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKLM "${UNINST_KEY}" "QuietUninstallString" '"$INSTDIR\uninstall.exe" /S'
  WriteRegStr HKLM "${UNINST_KEY}" "Installer" "nsis-${ARCH}"
  WriteRegDWORD HKLM "${UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKLM "${UNINST_KEY}" "NoRepair" 1
SectionEnd

Section "Add the orcaton command to PATH" SecPath
SectionEnd

Section "Desktop shortcut" SecDesktop
SectionEnd

Section "-Finish"
  ${If} ${SectionIsSelected} ${SecPath}
    !insertmacro RUN_PATH_TOOL Add Machine "$INSTDIR" ""
    ${If} $R0 != 0
      DetailPrint "PATH was not changed (code $R0)."
      MessageBox MB_OK|MB_ICONEXCLAMATION "Could not add the orcaton command to PATH.$\r$\n${APP} itself works; the command can be run as:$\r$\n$INSTDIR\orcaton.exe" /SD IDOK
    ${EndIf}
    WriteRegDWORD HKLM "${UNINST_KEY}" "OptPath" 1
  ${Else}
    !insertmacro RUN_PATH_TOOL Remove Machine "$INSTDIR" ""
    WriteRegDWORD HKLM "${UNINST_KEY}" "OptPath" 0
  ${EndIf}
  ${If} ${SectionIsSelected} ${SecDesktop}
    CreateShortcut "$DESKTOP\${APP}.lnk" "$INSTDIR\${EXE}" "" "$INSTDIR\${EXE}" 0 SW_SHOWNORMAL "" "${APP} — multicam audio sync"
    WriteRegDWORD HKLM "${UNINST_KEY}" "OptDesktop" 1
  ${Else}
    Delete "$DESKTOP\${APP}.lnk"
    WriteRegDWORD HKLM "${UNINST_KEY}" "OptDesktop" 0
  ${EndIf}
  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  WriteRegDWORD HKLM "${UNINST_KEY}" "EstimatedSize" "$0"
  SetOutPath "$INSTDIR"
  Call CheckWebView2
SectionEnd

Function RemoveUserCopy
  ${If} $OldDir == $INSTDIR
    Return
  ${EndIf}
  ClearErrors
  ReadRegStr $R1 HKCU "${UNINST_KEY}" "DisplayName"
  ${IfNot} ${FileExists} "$OldDir\${EXE}"
  ${AndIf} $R1 == ""
    Return
  ${EndIf}
  DetailPrint "Removing the earlier per-user copy: $OldDir"
  StrCpy $R6 "$OldDir"
  Call CheckRunning
  ${If} $R9 == 1
    Abort "${APP} (per-user copy) is running — installation cancelled."
  ${EndIf}
  Delete "$OldDir\${EXE}"
  Delete "$OldDir\orcaton.exe"
  Delete "$OldDir\LICENSE"
  Delete "$OldDir\NOTICE"
  Delete "$OldDir\THIRD-PARTY-NOTICES.md"
  Delete "$OldDir\VERSION"
  Delete "$OldDir\uninstall.exe"
  Delete "$OldDir\uninstall.ps1"
  Delete "$OldDir\install-manifest.txt"
  RMDir "$OldDir"
  SetShellVarContext current
  Delete "$SMPROGRAMS\${APP}.lnk"
  Delete "$DESKTOP\${APP}.lnk"
  StrCpy $R2 "$LOCALAPPDATA\Programs\orcaton-go"
  SetShellVarContext all
  !insertmacro RUN_PATH_TOOL Remove User "$OldDir" '-Also "$R2"'
  DeleteRegKey HKCU "${UNINST_KEY}"
FunctionEnd

Function DirectoryLeave
  StrLen $0 "$INSTDIR"
  ${If} $0 <= 3
    MessageBox MB_OK|MB_ICONEXCLAMATION "Choose a folder, not the root of a drive (for example C:\Program Files\Orcaton)."
    Abort
  ${EndIf}
  StrCpy $R6 "$INSTDIR"
  Call CheckRunning
  ${If} $R9 == 1
    Abort
  ${EndIf}
FunctionEnd

Function RunApp
  Exec '"$WINDIR\explorer.exe" "$INSTDIR\${EXE}"'
FunctionEnd

Function CheckWebView2
  StrCpy $1 ""
  ReadRegStr $1 HKLM "SOFTWARE\Microsoft\EdgeUpdate\Clients\${WEBVIEW2_ID}" "pv"
  ${If} $1 == ""
  ${OrIf} $1 == "0.0.0.0"
    SetRegView 32
    ReadRegStr $1 HKLM "SOFTWARE\Microsoft\EdgeUpdate\Clients\${WEBVIEW2_ID}" "pv"
    SetRegView 64
  ${EndIf}
  ${If} $1 == ""
  ${OrIf} $1 == "0.0.0.0"
    ReadRegStr $1 HKCU "SOFTWARE\Microsoft\EdgeUpdate\Clients\${WEBVIEW2_ID}" "pv"
  ${EndIf}
  ${If} $1 == ""
  ${OrIf} $1 == "0.0.0.0"
    DetailPrint "Microsoft Edge WebView2 Runtime was not found."
    MessageBox MB_YESNO|MB_ICONINFORMATION "${APP} needs Microsoft Edge WebView2 Runtime (included in Windows 11), and it was not found.$\r$\n$\r$\nOpen the Microsoft download page now?" /SD IDNO IDNO +2
    ExecShell "open" "${WEBVIEW2_URL}"
  ${Else}
    DetailPrint "WebView2 Runtime: $1"
  ${EndIf}
FunctionEnd

Function .onInit
  ${IfNot} ${AtLeastBuild} 10240
    MessageBox MB_OK|MB_ICONSTOP "${APP} needs Windows 10 or newer." /SD IDOK
    Abort
  ${EndIf}
!if "${ARCH}" == "arm64"
  ${IfNot} ${IsNativeARM64}
    MessageBox MB_OK|MB_ICONSTOP "This is the ARM64 installer of ${APP}.$\r$\nThis PC needs the x64 installer: Orcaton-${VERSION}-windows-x64-setup.exe" /SD IDOK
    Abort
  ${EndIf}
!else
  ${IfNot} ${RunningX64}
    MessageBox MB_OK|MB_ICONSTOP "${APP} needs 64-bit Windows." /SD IDOK
    Abort
  ${EndIf}
  ${If} ${IsNativeARM64}
    MessageBox MB_OKCANCEL|MB_ICONINFORMATION "This is the x64 installer; it works on this ARM PC through emulation.$\r$\nThe ARM64 installer (Orcaton-${VERSION}-windows-arm64-setup.exe) is faster.$\r$\n$\r$\nContinue with x64?" /SD IDOK IDOK +2
    Abort
  ${EndIf}
!endif
  Call IsAdmin
  SetRegView 64
  SetShellVarContext current
  StrCpy $OldDir "$LOCALAPPDATA\Programs\Orcaton"
  SetShellVarContext all
  ${If} $INSTDIR == "$PROGRAMFILES64\Orcaton"
    ReadRegStr $0 HKLM "${UNINST_KEY}" "InstallLocation"
    ${If} $0 != ""
      StrCpy $INSTDIR $0
    ${EndIf}
  ${EndIf}
  ClearErrors
  ReadRegDWORD $0 HKLM "${UNINST_KEY}" "OptPath"
  ${IfNot} ${Errors}
  ${AndIf} $0 == 0
    !insertmacro UnselectSection ${SecPath}
  ${EndIf}
  ClearErrors
  ReadRegDWORD $0 HKLM "${UNINST_KEY}" "OptDesktop"
  ${IfNot} ${Errors}
  ${AndIf} $0 == 0
    !insertmacro UnselectSection ${SecDesktop}
  ${EndIf}
FunctionEnd


Section "un.${APP}" UnSecMain
  SectionIn RO
  StrCpy $R6 "$INSTDIR"
  Call un.CheckRunning
  ${If} $R9 == 1
    Abort "${APP} is running — uninstall cancelled."
  ${EndIf}
  !insertmacro RUN_PATH_TOOL Remove Machine "$INSTDIR" ""
  ${If} $R0 != 0
    DetailPrint "PATH was not changed (code $R0); remove $INSTDIR from the system PATH manually if it is there."
  ${EndIf}
  Delete "$SMPROGRAMS\${APP}.lnk"
  Delete "$DESKTOP\${APP}.lnk"
  Delete "$INSTDIR\${EXE}"
  Delete "$INSTDIR\orcaton.exe"
  Delete "$INSTDIR\LICENSE"
  Delete "$INSTDIR\NOTICE"
  Delete "$INSTDIR\THIRD-PARTY-NOTICES.md"
  Delete "$INSTDIR\VERSION"
  Delete "$INSTDIR\uninstall.exe"
  SetOutPath "$TEMP"
  RMDir "$INSTDIR"
  DeleteRegKey HKLM "${UNINST_KEY}"
SectionEnd

Section /o "un.Also remove settings and cache" UnSecData
  SetShellVarContext current
  RMDir /r "$APPDATA\orcaton"
  RMDir /r "$LOCALAPPDATA\orcaton"
  RMDir /r "$APPDATA\${EXE}"
  DetailPrint "Settings, audio cache, logs and window data removed: $APPDATA\orcaton, $LOCALAPPDATA\orcaton, $APPDATA\${EXE}"
  SetShellVarContext all
SectionEnd

Section "-un.Done"
  ${IfNot} ${SectionIsSelected} ${UnSecData}
    DetailPrint "Settings and cache kept (per user: %APPDATA%\orcaton, %LOCALAPPDATA%\orcaton, %APPDATA%\${EXE})."
  ${EndIf}
  DetailPrint "FFmpeg and WebView2 are not touched."
SectionEnd

Function un.onInit
  Call un.IsAdmin
  SetRegView 64
  SetShellVarContext all
  ${GetParameters} $0
  ClearErrors
  ${GetOptions} $0 "/PURGE" $1
  ${IfNot} ${Errors}
    !insertmacro SelectSection ${UnSecData}
  ${EndIf}
FunctionEnd
