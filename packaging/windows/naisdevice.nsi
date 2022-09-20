; Definitions

!define APP_NAME "naisdevice"
!define VERSION "develop" ; Override when building release, MUST match '\d+.\d+.\d+.\d+'
!define UNINSTALLER "uninstaller.exe"
!define SOURCE "../../bin/windows-client"

; Includes ---------------------------------
!include MUI2.nsh
!include LogicLib.nsh

; Settings ---------------------------------
Name "${APP_NAME}"
OutFile "${APP_NAME}-${VERSION}.exe"
RequestExecutionLevel user
InstallDir "$PROGRAMFILES64\${APP_NAME}"

SetCompressor /SOLID lzma
AllowSkipFiles off

; File properties details
!if ${VERSION} != "develop"
VIAddVersionKey "ProductName" "${APP_NAME}"
VIAddVersionKey "CompanyName" "NAV - nais"
VIAddVersionKey "ProductVersion" "${VERSION}"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "FileDescription" "${APP_NAME}"
VIAddVersionKey "LegalCopyright" "NAV - nais"
VIProductVersion "${VERSION}"
!endif

; Pages ------------------------------------

!insertmacro MUI_PAGE_INSTFILES
; TODO: Add uninstaller pages

; Languages --------------------------------

!insertmacro MUI_LANGUAGE "English"

; Sections ---------------------------------

Section "-install files"
    CreateDirectory $INSTDIR
    SetOutPath $INSTDIR
    File ${SOURCE}\naisdevice-agent.exe
    File ${SOURCE}\naisdevice-systray.exe
    File ${SOURCE}\naisdevice-helper.exe
SectionEnd

Section "-create helper service"
SectionEnd

Section "-uninstaller"
    WriteUninstaller $INSTDIR\${UNINSTALLER}
SectionEnd

Section "Uninstall"
  ; TODO: Do proper cleanup
  Delete $INSTDIR\${UNINSTALLER} ; delete self (see explanation below why this works)
  Delete $INSTDIR\naisdevice-*.exe ; delete self (see explanation below why this works)
  RMDir $INSTDIR
SectionEnd

; Functions --------------------------------
