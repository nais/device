; Definitions

!define APP_NAME "naisdevice"
!define VERSION "develop" ; Override when building release, MUST match '\d+.\d+.\d+.\d+'
!define UNINSTALLER "uninstaller.exe"
!define SOURCE "../../bin/windows-client"
!define WIREGUARD "wireguard-amd64-0.5.3.msi"

; Includes ---------------------------------
!include MUI2.nsh
!include nsDialogs.nsh
!include LogicLib.nsh

; Settings ---------------------------------
Name "${APP_NAME}"
OutFile "${APP_NAME}-${VERSION}.exe"
RequestExecutionLevel user
InstallDir "$PROGRAMFILES64\NAV\${APP_NAME}"
AllowSkipFiles off

; File properties details
!if ${VERSION} != "develop"
SetCompressor /SOLID lzma
VIAddVersionKey "ProductName" "${APP_NAME}"
VIAddVersionKey "CompanyName" "NAV"
VIAddVersionKey "ProductVersion" "${VERSION}"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "FileDescription" "${APP_NAME}"
VIAddVersionKey "LegalCopyright" "NAV - nais"
VIProductVersion "${VERSION}"
!endif

; MUI settings

; Pages ------------------------------------

;; Installer pages

!insertmacro MUI_PAGE_WELCOME
; TODO: Add downgrade check
; TODO: Stop running instances of naisdevice *and* WireGuard
!insertmacro MUI_PAGE_INSTFILES

Var Dialog
Var Label
Page custom InstallWireGuard

;; Uninstaller pages

!insertmacro MUI_UNPAGE_WELCOME
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

; Languages --------------------------------

!insertmacro MUI_LANGUAGE "English"

; Sections ---------------------------------

Section "-install files"
    CreateDirectory $INSTDIR
    SetOutPath $INSTDIR
    File ${SOURCE}\naisdevice-*.exe
    File assets\naisdevice.ico
SectionEnd

Section "-create shortcuts"
    ; TODO
SectionEnd

Section "-create helper service"
    ; TODO
SectionEnd

Section "-create app_data folder (and logs)"
    ; TODO
SectionEnd

Section "-uninstaller"
    WriteUninstaller $INSTDIR\${UNINSTALLER}
SectionEnd

Section "-add to add/remove"
    ; TODO
SectionEnd

Section "Uninstall"
  ; TODO: Do proper cleanup
  Delete $INSTDIR\${UNINSTALLER} ; delete self (see explanation below why this works)
  Delete $INSTDIR\naisdevice-*.exe ; delete self (see explanation below why this works)
  RMDir $INSTDIR
SectionEnd

; Functions --------------------------------

Function InstallWireGuard
    !define header "Installation almost complete"
    !define subheader "Installing WireGuard"
    !define main_text "Installation of naisdevice is almost finished.$\n$\n\
                       The final step is to install WireGuard, which is used by naisdevice to create the VPN tunnels.$\n$\n\
                       The WireGuard installer finishes by launching WireGuard. You can close that window without making any changes.$\n$\n\
                       Have a nais day!"
    !insertmacro MUI_HEADER_TEXT "${header}" "${subheader}"

    nsDialogs::Create 1018
	Pop $Dialog
	${If} $Dialog == error
		Abort
	${EndIf}
	; Build info page
	${NSD_CreateLabel} 0 0 100% 100% "${main_text}"
    Pop $Label
	nsDialogs::Show

    SetOutPath $TEMP
    File "${WIREGUARD}"
    ExecWait 'msiexec /package "$TEMP\${WIREGUARD}"'
    IfErrors error_installing_dotnet
    Delete $TEMP\${WIREGUARD}
    Goto done
    error_installing_dotnet:
        Delete $TEMP\${WIREGUARD}
        Abort
    done:
FunctionEnd
