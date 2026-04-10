
### TimeStamp
!ifndef TimeStamp
    !define TimeStamp "!insertmacro _TimeStamp"
    !macro _TimeStamp FormatedString
        !ifdef __UNINSTALL__
            Call un.__TimeStamp
        !else
            Call __TimeStamp
        !endif
        Pop ${FormatedString}
    !macroend

!macro __TimeStamp UN
Function ${UN}__TimeStamp
    ClearErrors
    ## Store the needed Registers on the stack
        Push $0 ; Stack $0
        Push $1 ; Stack $1 $0
        Push $2 ; Stack $2 $1 $0
        Push $3 ; Stack $3 $2 $1 $0
        Push $4 ; Stack $4 $3 $2 $1 $0
        Push $5 ; Stack $5 $4 $3 $2 $1 $0
        Push $6 ; Stack $6 $5 $4 $3 $2 $1 $0
        Push $7 ; Stack $7 $6 $5 $4 $3 $2 $1 $0
        ;Push $8 ; Stack $8 $7 $6 $5 $4 $3 $2 $1 $0

    ## Call System API to get the current system Time
        System::Alloc 16
        Pop $0
        System::Call 'kernel32::GetLocalTime(i) i(r0)'
        System::Call '*$0(&i2, &i2, &i2, &i2, &i2, &i2, &i2, &i2)i (.r1, .r2, n, .r3, .r4, .r5, .r6, .r7)'
        System::Free $0

        IntFmt $2 "%02i" $2
        IntFmt $3 "%02i" $3
        IntFmt $4 "%02i" $4
        IntFmt $5 "%02i" $5
        IntFmt $6 "%02i" $6
        IntFmt $7 "%03i" $7

    ## Generate Timestamp
        StrCpy $0 "$1-$2-$3 $4:$5:$6.$7"

    ## Restore the Registers and add Timestamp to the Stack
        ;Pop $8  ; Stack $7 $6 $5 $4 $3 $2 $1 $0
        Pop $7  ; Stack $6 $5 $4 $3 $2 $1 $0
        Pop $6  ; Stack $5 $4 $3 $2 $1 $0
        Pop $5  ; Stack $4 $3 $2 $1 $0
        Pop $4  ; Stack $3 $2 $1 $0
        Pop $3  ; Stack $2 $1 $0
        Pop $2  ; Stack $1 $0
        Pop $1  ; Stack $0
        Exch $0 ; Stack ${TimeStamp}

FunctionEnd
!macroend
!insertmacro __TimeStamp ""
!insertmacro __TimeStamp "un."
!endif
###########


## Logging
!ifmacrondef _Log

Var _Log_FileHandle
Var _Log_Timestamp
Var _Log_Path
Var _Log_ProgramDataPath
Var _Log_EnvPath

!macro _Log text
    StrCpy $_Log_FileHandle ""

    ${If} $_Log_Path == ""
        ReadEnvStr $_Log_EnvPath "NAISDEVICE_INSTALL_LOG"
        ${If} $_Log_EnvPath != ""
            StrCpy $_Log_Path "$_Log_EnvPath"
        ${Else}
            GetKnownFolderPath $_Log_ProgramDataPath "${FOLDERID_ProgramData}"
            CreateDirectory "$_Log_ProgramDataPath\NAV"
            CreateDirectory "$_Log_ProgramDataPath\NAV\naisdevice"
            CreateDirectory "$_Log_ProgramDataPath\NAV\naisdevice\logs"
            StrCpy $_Log_Path "$_Log_ProgramDataPath\NAV\naisdevice\logs\installer.log"
        ${EndIf}
    ${EndIf}

    FileOpen $_Log_FileHandle "$_Log_Path" a

    ${If} $_Log_FileHandle == ""
        StrCpy $_Log_Path "$TEMP\nsis.log"
        FileOpen $_Log_FileHandle "$_Log_Path" a
    ${EndIf}

    ${If} $_Log_FileHandle != ""
        FileSeek $_Log_FileHandle 0 END
        ${TimeStamp} $_Log_Timestamp
        FileWrite $_Log_FileHandle "$_Log_Timestamp: ${text}$\n"
        FileClose $_Log_FileHandle
    ${EndIf}

    ${TimeStamp} $_Log_Timestamp
    DetailPrint "$_Log_Timestamp: ${text}"
!macroend

!endif