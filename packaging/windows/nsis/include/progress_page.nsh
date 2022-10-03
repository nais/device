
!include LogicLib.nsh
!include nsDialogs.nsh
!include utils.nsh

!ifndef ProgressPage
!define ProgressPage "!insertmacro _ProgressPage"

!ifdef __UNINSTALL__
    !define prefix "un."
!else
    !define prefix ""
!endif

!define __PP__PBS_MARQUEE 0x08
!define __PP__PAGE_TIMEOUT 60000
!define __PP__STEP_INTERVAL 100

Var __PP__ResultCode
Var __PP__Dialog
Var __PP__ProgressBar
Var __PP__Label
Var __PP__Timeout

Function ${prefix}PP_NoOp
FunctionEnd

; Usage:
; name - Generated function name
; header - Header of the page
; subheader - Sub-header of the page
; text - The descriptive text on the page, below the progressbar
; abort_cb - Function that should push 0 on the stack to skip the page, any other value to continue (required)
; init_cb - Function to initialize any needed state, PP_NoOp to skip
; step_cb - Function called on every step. Should push 0 to the stack to leave the page, any other value to continue (required)
; leaving_cb - Function called when successfully leaving the page, PP_NoOp to skip
!macro _ProgressPage name header subheader text abort_cb init_cb step_cb leaving_cb

Function ${prefix}${name}
    !insertmacro _Log "${prefix}${name} entered."

    Call ${prefix}${abort_cb}
    Pop $__PP__ResultCode
    ${If} $__PP__ResultCode = 0
        !insertmacro _Log "${prefix}${name}: Aborting page"
        Abort
        Return
    ${EndIf}

    !insertmacro _Log "${prefix}${name}: Showing page"

    !insertmacro MUI_HEADER_TEXT "${header}" "${subheader}"

    nsDialogs::Create 1018
	Pop $__PP__Dialog
	${If} $__PP__Dialog == error
		Abort
		Return
	${EndIf}

    ${NSD_CreateProgressBar} 0 0 100% 10% "Test"
    Pop $__PP__ProgressBar
	${If} $__PP__ProgressBar == error
		Abort
		Return
	${EndIf}

    ${NSD_AddStyle} $__PP__ProgressBar ${__PP__PBS_MARQUEE}

    EnableWindow $mui.Button.Next 0
	${NSD_CreateLabel} 0 15% 100% 100% "${text}"
    Pop $__PP__Label

    StrCpy $__PP__Timeout ${PAGE_TIMEOUT}
    ${NSD_CreateTimer} ${prefix}${name}__ProgressStepCallback ${STEP_INTERVAL}
	nsDialogs::Show
FunctionEnd

Function ${prefix}${name}__ProgressStepCallback
    !insertmacro _Log "${prefix}${name}__ProgressStepCallback entered. __PP__Timeout=$__PP__Timeout"

    Call ${prefix}${step_cb}
    Pop $__PP__ResultCode
    ${If} $__PP__ResultCode = 0
        !insertmacro _Log "${prefix}${name}__ProgressStepCallback: Progress completed. __PP__Timeout=$__PP__Timeout"
        IntOp $__PP__Timeout $__PP__Timeout - ${PAGE_TIMEOUT}
        !insertmacro _Log "${prefix}${name}__ProgressStepCallback: Cleared timeout. __PP__Timeout=$__PP__Timeout"
    ${EndIf}

    ${If} $__PP__Timeout = ${PAGE_TIMEOUT}
        ; Start progressbar and attempt stopping the service
        !insertmacro _Log "${prefix}${name}__ProgressStepCallback: Starting progressbar. __PP__Timeout=$__PP__Timeout"
        SendMessage $__PP__ProgressBar ${PBM_SETMARQUEE} 1 50 ; start=1|stop=0 interval(ms)=+N
        !insertmacro _Log "${prefix}${name}__ProgressStepCallback: Calling init_cb"
        Call ${prefix}${init_cb}
    ${ElseIf} $__PP__Timeout <= 0
        ; Timeout ended, clear progressbar and progress to next page
        !insertmacro _Log "${prefix}${name}__ProgressStepCallback: Timeout ended, killing timer and resetting progress. __PP__Timeout=$__PP__Timeout"
        ${NSD_KillTimer} ${prefix}${name}__ProgressStepCallback
        SendMessage $__PP__ProgressBar ${PBM_SETMARQUEE} 0 0 ; start=1|stop=0 interval(ms)=+N
        !insertmacro _Log "${prefix}${name}__ProgressStepCallback: Calling leaving_cb"
        Call ${prefix}${leaving_cb}
        !insertmacro _Log "${prefix}${name}__ProgressStepCallback: Enabling and clicking next button"
        EnableWindow $mui.Button.Next 1
        SendMessage $mui.Button.Next ${BM_CLICK} 0 0
    ${EndIf}
    IntOp $__PP__Timeout $__PP__Timeout - ${STEP_INTERVAL}
    !insertmacro _Log "${prefix}${name}__ProgressStepCallback: Leaving. __PP__Timeout=$__PP__Timeout"
FunctionEnd

!macroend
!endif
