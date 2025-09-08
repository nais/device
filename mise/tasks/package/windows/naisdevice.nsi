; Additional includes and plugins

!addincludedir "nsis\include"
!addplugindir "nsis\plugins"

; Definitions

!define /ifndef VERSION "develop" ; Override when building release, MUST match '\d+.\d+.\d+.\d+'

!define APP_NAME "naisdevice"
!define UNINSTALLER "uninstaller.exe"
!define SOURCE "./bin/windows-client"
!define WIREGUARD "wireguard-amd64-0.5.3.msi"
!define REG_UNINSTALL "SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall"
!define REG_ARP "${REG_UNINSTALL}\${APP_NAME}"
!define SERVICE_NAME "NaisDeviceHelper"
!define PAGE_TIMEOUT 60000
!define STEP_INTERVAL 100

; Microsoft defined IDs not available in headers
!define FOLDERID_ProgramData "{62AB5D82-FDC1-4DC3-A9DD-070D1D495D97}"
!define PBS_MARQUEE 0x08
!define SERVICE_WIN32_OWN_PROCESS 16
!define SERVICE_AUTO_START 2

; Includes ---------------------------------

!if ${VERSION} != "develop"
SetCompressor /SOLID lzma
!endif

!include WinCore.nsh
!include MUI2.nsh
!include nsDialogs.nsh
!include LogicLib.nsh
!include FileFunc.nsh
!include nsProcess.nsh
!include utils.nsh
!include progress_page.nsh

; Settings ---------------------------------
Name "${APP_NAME}"
OutFile "${APP_NAME}.exe"
RequestExecutionLevel admin
InstallDir "$PROGRAMFILES64\NAV\${APP_NAME}"
AllowSkipFiles off

; File properties details
!if ${VERSION} != "develop"
VIAddVersionKey "ProductName" "${APP_NAME}"
VIAddVersionKey "CompanyName" "NAV"
VIAddVersionKey "ProductVersion" "${VERSION}"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "FileDescription" "${APP_NAME}"
VIAddVersionKey "LegalCopyright" "NAV - nais"
VIProductVersion "${VERSION}"
!endif

; Configure signing if certificates available
!ifdef CERT_FILE & KEY_FILE
!finalize './sign-exe "%1" "${CERT_FILE}" "${KEY_FILE}"' = 0
!uninstfinalize './sign-exe "%1" "${CERT_FILE}" "${KEY_FILE}"' = 0
!endif

; Global variables :scream:

Var Result
Var ProgramDataPath

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
UninstPage custom un.StopInstances
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

; Languages --------------------------------

!insertmacro MUI_LANGUAGE "English"

; Sections ---------------------------------

Section "Uninstall legacy version"
    StrCpy $0 0
    loop:
      EnumRegKey $1 HKLM "${REG_UNINSTALL}" $0
      IntOp $0 $0 + 1
      StrCmp $1 "" done
      StrCmp $1 "naisdevice" loop
      ReadRegStr $2 HKLM "${REG_UNINSTALL}\$1" "DisplayName"
      ${If} $2 == "naisdevice"
        !insertmacro _Log "Found add/remove entry for legacy uninstaller"
        ExecWait 'MsiExec.exe /uninstall $1 /qn'
      ${EndIf}
      Goto loop
    done:
SectionEnd

Section "Install files"
    CreateDirectory $INSTDIR
    SetOutPath $INSTDIR
    File ${SOURCE}\naisdevice-*.exe
    File assets\naisdevice.ico
SectionEnd

Section "Create data folder"
    GetKnownFolderPath $ProgramDataPath "${FOLDERID_ProgramData}"
    CreateDirectory "$ProgramDataPath\NAV\naisdevice\etc"
    CreateDirectory "$ProgramDataPath\NAV\naisdevice\logs"
    CreateDirectory "$ProgramDataPath\NAV\naisdevice\run"

    AccessControl::GrantOnFile "$ProgramDataPath\NAV\naisdevice" "(BU)" "FullAccess"
    Pop $R0
    ${If} $R0 == error
        Pop $R0
        !insertmacro _Log "Failed to grant permissions to data folder: $R0"
    ${EndIf}
SectionEnd

Section "Create shortcuts"
    GetKnownFolderPath $ProgramDataPath "${FOLDERID_ProgramData}"
    SetOutPath "$ProgramDataPath\NAV\naisdevice"
    CreateShortcut "$SMPROGRAMS\naisdevice.lnk" "$INSTDIR\naisdevice-systray.exe" "" "$INSTDIR\naisdevice.ico" \
        "" "" "" "naisdevice is a mechanism enabling NAVs developers to connect to internal resources in a secure and friendly manner"
    CreateShortcut "$SMSTARTUP\naisdevice.lnk" "$INSTDIR\naisdevice-systray.exe" "" "$INSTDIR\naisdevice.ico" \
        "" "" "" "naisdevice is a mechanism enabling NAVs developers to connect to internal resources in a secure and friendly manner"
SectionEnd

Section "Create helper service"
    ; Install service
    !insertmacro _Log "Installing NaisDeviceHelper service"
    SimpleSC::InstallService ${SERVICE_NAME} "naisdevice helper" ${SERVICE_WIN32_OWN_PROCESS} ${SERVICE_AUTO_START} \
        '"$INSTDIR\naisdevice-helper.exe" --interface utun69' "" "NT AUTHORITY\SYSTEM"
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
    WriteRegStr HKLM "${REG_ARP}" "HelpLink" "https://doc.nais.io/operate/naisdevice"
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
    RMDir /r "$ProgramDataPath\NAV\naisdevice"
    Delete $INSTDIR\naisdevice-*.exe
    Delete $INSTDIR\naisdevice.ico
    Delete $INSTDIR\${UNINSTALLER}
    RMDir $INSTDIR
    !insertmacro _Log "Cleaning up registry"
    DeleteRegKey HKLM "${REG_ARP}"
SectionEnd

; Functions --------------------------------

!macro GUIInit un
Function ${un}GUIInit
    !insertmacro _Log "Inside GUIInit"
    SetRegView 64
    SetShellVarContext all
FunctionEnd
!macroend
!insertmacro GUIInit ""
!insertmacro GUIInit "un."


; Places number of running instances on stack
!macro CountRunningInstances un
Function ${un}CountRunningInstances
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
!macroend
!insertmacro CountRunningInstances ""
!insertmacro CountRunningInstances "un."

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

; -- StopInstances --------------

!macro StopInstances action prefix
; Function that should push 0 on the stack to skip the page, any other value to continue (required)
Function ${prefix}_StopInstances_Abort
    Push $0
    Push $1

    SimpleSC::ExistsService ${SERVICE_NAME} ; <> 0 => service exists
    Pop $0
    !insertmacro _Log "Service exists: $0 (0: true, *: false)"
    Call ${prefix}CountRunningInstances
    Pop $1
    !insertmacro _Log "Number of running instances: $1"
    ${If} $0 <> 0
    ${AndIf} $1 = 0
        !insertmacro _Log "Skipping stop instances"
        Push 0
    ${Else}
        Push 1
    ${EndIf}

    Exch 2
    Pop $0
    Pop $1
FunctionEnd

; Function to initialize any needed state, "" to skip
Function ${prefix}_StopInstances_Init
    !insertmacro _Log "Attempting to stop ${SERVICE_NAME}"
    SimpleSC::StopService ${SERVICE_NAME} 1 40
    !insertmacro _Log "StopService ${SERVICE_NAME} completed"
FunctionEnd

; Function called on every step. Should push 0 to the stack to leave the page, any other value to continue (required)
Function ${prefix}_StopInstances_Step
    Push $0

    !insertmacro _Log "Attempt to stop running processes"
    Call ${prefix}CloseRunningInstances

    Call ${prefix}CountRunningInstances
    Pop $0
    ${If} $0 = 0
        ; No more processes, clear timeout ending loop
        !insertmacro _Log "No more processes"
        Push 0
    ${Else}
        Push 1
    ${EndIf}

    Exch
    Pop $0
FunctionEnd

${ProgressPage} \
    "${prefix}StopInstances" \
    "Stopping running instances" \
    " " \
    "Before we can ${action} this version of naisdevice we need to stop any running instances.$\n$\n\
        This includes stopping the NaisDeviceHelper service running in the background.$\n$\n\
        Unfortunately, this may take some time, so please be patient while we attempt to stop everything." \
    _StopInstances_Abort \
    _StopInstances_Init \
    _StopInstances_Step \
    PP_NoOp

!macroend
!insertmacro StopInstances "install" ""
!insertmacro StopInstances "uninstall" "un."

; -- InstallWireGuard --------------

; Function that should push 0 on the stack to skip the page, any other value to continue (required)
Function _InstallWireGuard_Abort
    ; Never skip
    Push 1
FunctionEnd

; Function to initialize any needed state, PP_NoOp to skip
Function _InstallWireGuard_Init
    Push $R9

    SetOutPath $TEMP
    File "${WIREGUARD}"
    ExecWait 'msiexec /package "$TEMP\${WIREGUARD}" DO_NOT_LAUNCH=true' $R9
    ${If} ${Errors}
        !insertmacro _Log "Error while installing WireGuard"
        !insertmacro _Log "Exit code from wireguard installer: $R9"
    ${EndIf}

    Pop $R9
FunctionEnd

; Function called on every step. Should push 0 to the stack to leave the page, any other value to continue (required)
Function _InstallWireGuard_Step
    Push $R9

    !insertmacro _Log "Attempting to start ${SERVICE_NAME} service"
    SimpleSC::StartService ${SERVICE_NAME} "" 60
    Pop $R9
    !insertmacro _Log "Result of starting service: $R9 (errorcode (<>0) otherwise success (0))"
    ${If} $R9 != 0
        Push $R9
        SimpleSC::GetErrorMessage
        Pop $R9
        !insertmacro _Log "Starting service failed: $R9"
        Push 1
    ${Else}
        Push 0
    ${EndIf}

    Exch
    Pop $R9
FunctionEnd

; Function called when successfully leaving the page, PP_NoOp to skip
Function _InstallWireGuard_Leaving
    Delete $TEMP\${WIREGUARD}
FunctionEnd

${ProgressPage} \
    "InstallWireGuard" \
    "Installation almost complete" \
    "Installing WireGuard and starting services" \
    "Installation of naisdevice is almost finished.$\n$\n\
        The final steps are to install WireGuard, which is used by naisdevice to create the VPN tunnels, and start background services.$\n$\n\
        The WireGuard installer finishes by launching WireGuard. You can close that window without making any changes.$\n$\n\
        Have a nais day!" \
    _InstallWireGuard_Abort \
    _InstallWireGuard_Init \
    _InstallWireGuard_Step \
    _InstallWireGuard_Leaving
