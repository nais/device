; Additional includes and plugins

!addincludedir "nsis\include"
!addplugindir "nsis\plugins"

; Definitions

!define APP_NAME "naisdevice"
!define VERSION "develop" ; Override when building release, MUST match '\d+.\d+.\d+.\d+'
!define UNINSTALLER "uninstaller.exe"
!define SOURCE "../../bin/windows-client"
!define WIREGUARD "wireguard-amd64-0.5.3.msi"
!define LEGACY_PRODUCT_CODE "{56053D33-DC41-43BC-99D0-C9569C306E79}"
!define REG_UNINSTALL "SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall"
!define REG_ARP "${REG_UNINSTALL}\${APP_NAME}"
!define REG_LEGACY "${REG_UNINSTALL}\${LEGACY_PRODUCT_CODE}"
!define SERVICE_NAME "NaisDeviceHelper"
!define PAGE_TIMEOUT 60000
!define STEP_INTERVAL 100

; Microsoft defined IDs not available in headers
!define FOLDERID_ProgramData "{62AB5D82-FDC1-4DC3-A9DD-070D1D495D97}"
!define PBS_MARQUEE 0x08
!define SERVICE_WIN32_OWN_PROCESS 16
!define SERVICE_AUTO_START 2

; Includes ---------------------------------
!include WinCore.nsh
!include MUI2.nsh
!include nsDialogs.nsh
!include LogicLib.nsh
!include FileFunc.nsh
!include nsProcess.nsh
!include utils.nsh

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

; Global variables :scream:

Var Dialog
Var ProgressBar
Var Label
Var Timeout
Var Result
Var ProgramDataPath
Var LegacyUninstallerCmd

; MUI Settings -----------------------------

!define MUI_CUSTOMFUNCTION_GUIINIT GUIInit
!define MUI_CUSTOMFUNCTION_UNGUIINIT un.GUIInit

; Pages ------------------------------------

;; Installer pages

!insertmacro MUI_PAGE_WELCOME
Page custom StopInstances
!insertmacro MUI_PAGE_INSTFILES
Page custom InstallWireGuard

;; Uninstaller pages

!insertmacro MUI_UNPAGE_WELCOME
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

; Languages --------------------------------

!insertmacro MUI_LANGUAGE "English"

; Sections ---------------------------------

Section "Uninstall legacy version"
    ReadRegStr $LegacyUninstallerCmd HKLM "${REG_LEGACY}" "UninstallString"
    ${If} $LegacyUninstallerCmd != ""
        !insertmacro _Log "Executing legacy uninstaller"
        ExecWait 'MsiExec.exe /uninstall ${LEGACY_PRODUCT_CODE} /qn'
    ${EndIf}
SectionEnd

Section "Install files"
    CreateDirectory $INSTDIR
    SetOutPath $INSTDIR
    File ${SOURCE}\naisdevice-*.exe
    File assets\naisdevice.ico
SectionEnd

Section "Create data folder"
    ; TODO: Check permissions on folders
    GetKnownFolderPath $ProgramDataPath "${FOLDERID_ProgramData}"
    CreateDirectory "$ProgramDataPath\NAV\naisdevice\etc"
    CreateDirectory "$ProgramDataPath\NAV\naisdevice\logs"
    CreateDirectory "$ProgramDataPath\NAV\naisdevice\run"
SectionEnd

Section "Create shortcuts"
    GetKnownFolderPath $ProgramDataPath "${FOLDERID_ProgramData}"
    SetOutPath "$ProgramDataPath\NAV\naisdevice"
    CreateShortcut "$SMPROGRAMS\naisdevice.lnk" "$INSTDIR\naisdevice-systray.exe" "" "$INSTDIR\naisdevice.ico" "" "" "" "naisdevice is a mechanism enabling NAVs developers to connect to internal resources in a secure and friendly manner."
    CreateShortcut "$SMSTARTUP\naisdevice.lnk" "$INSTDIR\naisdevice-systray.exe" "" "$INSTDIR\naisdevice.ico" "" "" "" "naisdevice is a mechanism enabling NAVs developers to connect to internal resources in a secure and friendly manner."
SectionEnd

Section "Create helper service"
    ; Install service
    !insertmacro _Log "Installing NaisDeviceHelper service"
    SimpleSC::InstallService ${SERVICE_NAME} "naisdevice helper" ${SERVICE_WIN32_OWN_PROCESS} ${SERVICE_AUTO_START} \
        '"$INSTDIR\naisdevice-helper.exe" --interface utun69' "WireGuardManager" "NT AUTHORITY\SYSTEM"
    SimpleSC::SetServiceDescription ${SERVICE_NAME} "Controls the WireGuard VPN connection"
SectionEnd

Section "-uninstaller"
    WriteUninstaller $INSTDIR\${UNINSTALLER}
SectionEnd

Section "-add to add/remove"
    ; Add simple details
    WriteRegStr HKLM "${REG_ARP}" "DisplayName" "${APP_NAME}"
    WriteRegStr HKLM "${REG_ARP}" "UninstallString" "$\"$INSTDIR\${UNINSTALLER}$\""
    WriteRegStr HKLM "${REG_ARP}" "QuietUninstallString" "$\"$INSTDIR\${UNINSTALLER}$\" /S"
    WriteRegStr HKLM "${REG_ARP}" "InstallLocation" "$INSTDIR"
    WriteRegStr HKLM "${REG_ARP}" "DisplayIcon" "$INSTDIR\naisdevice.ico"
    WriteRegStr HKLM "${REG_ARP}" "ProductID" ""
    WriteRegStr HKLM "${REG_ARP}" "HelpLink" "https://doc.nais.io/device"
    WriteRegStr HKLM "${REG_ARP}" "URLUpdateInfo" "https://github.com/nais/device/releases/latest"
    WriteRegStr HKLM "${REG_ARP}" "URLInfoAbout" "slack://channel?team=T5LNAMWNA&amp;id=D011T20LDHD"
    WriteRegStr HKLM "${REG_ARP}" "DisplayVersion" "${VERSION}"

    WriteRegDWORD HKLM "${REG_ARP}" "NoModify" "1"
    WriteRegDWORD HKLM "${REG_ARP}" "NoRepair" "1"

    ; Add estimated size
    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD HKLM "${REG_ARP}" "EstimatedSize" "$0"
SectionEnd

Section "Uninstall"
    !insertmacro _Log "Stopping background service"
    SimpleSC::StopService ${SERVICE_NAME} 1 60
    !insertmacro _Log "Stopping running instances"
    Call un.CloseRunningInstances
    !insertmacro _Log "Removing background service"
    SimpleSC::RemoveService ${SERVICE_NAME}
    !insertmacro _Log "Deleting shortcuts"
    Delete "$SMPROGRAMS\naisdevice.lnk"
    Delete "$SMSTARTUP\naisdevice.lnk"
    !insertmacro _Log "Removing installed files"
    GetKnownFolderPath $ProgramDataPath "${FOLDERID_ProgramData}"
    RMDir /r "$ProgramDataPath\NAV"
    Delete $INSTDIR\naisdevice-*.exe
    Delete $INSTDIR\naisdevice.ico
    Delete $INSTDIR\${UNINSTALLER}
    RMDir $INSTDIR
    !insertmacro _Log "Cleaning up registry"
    DeleteRegKey HKLM "${REG_ARP}"
SectionEnd

; Functions --------------------------------

Function GUIInit
    !insertmacro _Log "Inside GUIInit"
    SetRegView 64
    SetShellVarContext all
FunctionEnd

Function un.GUIInit
    !insertmacro _Log "Inside un.GUIInit"
    SetRegView 64
    SetShellVarContext all
FunctionEnd

Function InstallWireGuard
    !insertmacro _Log "InstallWireGuard entered."

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
    ExecWait 'msiexec /package "$TEMP\${WIREGUARD}"' $R9
    ${If} ${Errors}
        !insertmacro _Log "Error while installing WireGuard"
        !insertmacro _Log "Exit code from wireguard installer: $R9"
    ${Else}
        !insertmacro _Log "Exit code from wireguard installer: $R9"
        Call StartService
    ${EndIf}
    Delete $TEMP\${WIREGUARD}
FunctionEnd

Function StartService
    !insertmacro _Log "StartService entered."

    Push $R9
    StrCpy $Timeout ${PAGE_TIMEOUT}
    ${Do}
        !insertmacro _Log "Attempting to start ${SERVICE_NAME} service"
        SimpleSC::StartService ${SERVICE_NAME} "" 60
        Pop $R9
        !insertmacro _Log "Result of starting service: $R9 (errorcode (<>0) otherwise success (0))"
        ${If} $R9 != 0
            Push $R9
            SimpleSC::GetErrorMessage
            Pop $0
            !insertmacro _Log "Starting service failed: $0"
        ${EndIf}
        IntOp $Timeout $Timeout - 100
        ${If} $Timeout < 0
            !insertmacro _Log "Timeout expired, breaking"
            ${Break}
        ${EndIf}
    ${LoopUntil} $R9 = 0
    Pop $R9
FunctionEnd

; Places number of running instances on stack
Function CountRunningInstances
    !insertmacro _Log "CountRunningInstances entered."

    Push $R9
    StrCpy $R9 0
    ${nsProcess::FindProcess} "naisdevice-agent.exe" $Result
    ${If} $Result = 0
        !insertmacro _Log "naisdevice-agent.exe still running"
        IntOp $R9 $R9 + 1
    ${EndIf}
    ${nsProcess::FindProcess} "naisdevice-systray.exe" $Result
    ${If} $Result = 0
        !insertmacro _Log "naisdevice-systray.exe still running"
        IntOp $R9 $R9 + 1
    ${EndIf}
    ${nsProcess::FindProcess} "naisdevice-helper.exe" $Result
    ${If} $Result = 0
        !insertmacro _Log "naisdevice-helper.exe still running"
        IntOp $R9 $R9 + 1
    ${EndIf}
    Push $R9
    Exch
    Pop $R9
FunctionEnd

; Places number of closed processes on stack
!macro CloseRunningInstances un
Function ${un}CloseRunningInstances
    !insertmacro _Log "${un}CloseRunningInstances entered."

    Push $R9
    StrCpy $R9 0
    ${nsProcess::CloseProcess} "naisdevice-agent.exe" $Result
    ${If} $Result = 0
        !insertmacro _Log "naisdevice-agent.exe closed"
        IntOp $R9 $R9 + 1
    ${EndIf}
    ${nsProcess::CloseProcess} "naisdevice-systray.exe" $Result
    ${If} $Result = 0
        !insertmacro _Log "naisdevice-systray.exe closed"
        IntOp $R9 $R9 + 1
    ${EndIf}
    ${nsProcess::CloseProcess} "naisdevice-helper.exe" $Result
    ${If} $Result = 0
        !insertmacro _Log "naisdevice-helper.exe closed"
        IntOp $R9 $R9 + 1
    ${EndIf}
    Push $R9
    Exch
    Pop $R9
FunctionEnd
!macroend
!insertmacro CloseRunningInstances ""
!insertmacro CloseRunningInstances "un."

Function StopInstances
    !insertmacro _Log "StopInstances entered."

    SimpleSC::ExistsService ${SERVICE_NAME} ; <> 0 => service exists
    Pop $0
    !insertmacro _Log "Service exists: $0 (0: true, *: false)"
    Call CountRunningInstances
    Pop $1
    !insertmacro _Log "Number of running instances: $1"
    ${If} $0 <> 0
    ${AndIf} $1 = 0
        !insertmacro _Log "Skipping stop instances"
        Abort
    ${EndIf}

    !insertmacro _Log "Stopping instances"

    !define ss_header "Stopping running instances"
    !define ss_subheader "Stopping previous version to allow overwriting files"
    !define ss_main_text "Before we can install this version of naisdevice we need to stop any running instances.$\n$\n\
                       This includes stopping the NaisDeviceHelper service running in the background.$\n$\n\
                       Unfortunately, this may take some time, so please be patient while we attempt to stop everything."
    !insertmacro MUI_HEADER_TEXT "${ss_header}" "${ss_subheader}"

    nsDialogs::Create 1018
	Pop $Dialog
	${If} $Dialog == error
		Abort
	${EndIf}

    ${NSD_CreateProgressBar} 0 0 100% 10% "Test"
    Pop $ProgressBar
	${If} $ProgressBar == error
		Abort
	${EndIf}

    ${NSD_AddStyle} $ProgressBar ${PBS_MARQUEE}

    EnableWindow $mui.Button.Next 0
	${NSD_CreateLabel} 0 15% 100% 100% "${ss_main_text}"
    Pop $Label

    StrCpy $Timeout ${PAGE_TIMEOUT}
    ${NSD_CreateTimer} ProgressStepCallback ${STEP_INTERVAL}
	nsDialogs::Show
FunctionEnd

Function ProgressStepCallback
    !insertmacro _Log "ProgressStepCallback entered. Timeout=$Timeout"

    Call CountRunningInstances
    Pop $0
    ${If} $0 = 0
        ; No more processes, clear timeout ending loop
        !insertmacro _Log "No more processes, clearing timeout. Timeout=$Timeout"
        IntOp $Timeout $Timeout - ${PAGE_TIMEOUT}
        !insertmacro _Log "Cleared timeout. Timeout=$Timeout"
    ${EndIf}

    ${If} $Timeout = ${PAGE_TIMEOUT}
        ; Start progressbar and attempt stopping the service
        !insertmacro _Log "Starting progressbar. Timeout=$Timeout"
        SendMessage $ProgressBar ${PBM_SETMARQUEE} 1 50 ; start=1|stop=0 interval(ms)=+N
        !insertmacro _Log "Attempting to stop ${SERVICE_NAME}. Timeout=$Timeout"
        SimpleSC::StopService ${SERVICE_NAME} 1 40
        !insertmacro _Log "Stopped ${SERVICE_NAME}. Timeout=$Timeout"
    ${ElseIf} $Timeout <= 0
        ; Timeout ended, clear progressbar and progress to next page
        !insertmacro _Log "Timeout ended, killing timer, resetting progress, enabling and clicking Next. Timeout=$Timeout"
        ${NSD_KillTimer} ProgressStepCallback
        SendMessage $ProgressBar ${PBM_SETMARQUEE} 0 0 ; start=1|stop=0 interval(ms)=+N
        EnableWindow $mui.Button.Next 1
        SendMessage $mui.Button.Next ${BM_CLICK} 0 0
    ${Else}
        ; Attempt to stop the processes forcefully
        !insertmacro _Log "Attempt to stop running processes. Timeout=$Timeout"
        Call CloseRunningInstances
    ${EndIf}
    IntOp $Timeout $Timeout - ${STEP_INTERVAL}
    !insertmacro _Log "Leaving ProgressStepCallback. Timeout=$Timeout"
FunctionEnd
