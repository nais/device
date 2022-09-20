; Definitions

!define APP_NAME "naisdevice"
!define VERSION "develop" ; Override when building release, MUST match '\d+.\d+.\d+.\d+'

; Includes ---------------------------------
!include MUI2.nsh
!include LogicLib.nsh

; Settings ---------------------------------
Name "${APP_NAME}"
OutFile "${APP_NAME}-${VERSION}.exe"
RequestExecutionLevel user
InstallDir "$PROGRAMFILES\${APP_NAME}"

SetCompressor /SOLID lzma
Icon "naisdevice.ico"
UninstallIcon "naisdevice.ico"
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

!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
; TODO: Add uninstaller pages

; Languages --------------------------------

!insertmacro MUI_LANGUAGE "English"

; Sections ---------------------------------

Section "-naisdevice agent"
MessageBox MB_OK "Installing agent"
SectionEnd

Section "-naisdevice systray"
MessageBox MB_OK "Installing systray"
SectionEnd

Section "-naisdevice helper"
MessageBox MB_OK "Installing helper"
SectionEnd

; Functions --------------------------------
