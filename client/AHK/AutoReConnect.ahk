#Requires AutoHotkey v2.0.18+ 64-bit

F12::ExitApp

Debug_Mode := true
LogFile := A_ScriptDir "\debug.log"

Log(msg, mode := "tip") {
    global Debug_Mode, LogFile
    if !Debug_Mode
        return

    timestamp := FormatTime(, "yyyy-MM-dd HH:mm:ss")
    fullMsg := "[" timestamp "] " msg

    switch mode {
        case "console":
            OutputDebug(fullMsg)
        case "file":
            FileAppend(fullMsg "`n", LogFile, "UTF-8")
        case "tip":
            ToolTip(fullMsg)
            ;SetTimer(() => ToolTip(), -1000)
        default:
            MsgBox(fullMsg)
    }
}

; -------------------------
gameTitle := "SCUM" A_Space A_Space
disconnectOK := "..\png\disconnect_ok.png"
continueBtn := "..\png\continue_game.png"
chatIcon    := "..\png\chat_icon.png"
chatGlobal := "..\png\chat_GlobalChannel.png"

chatColorX := 365
chatColorY := 307
;chatBlue   := 0x122E34

loadCheckX := 58
loadCheckY := 68
loadCheckColor := 0xFFFFFF

STATE_NORMAL    := 0
STATE_DISCONNECT:= 1
STATE_MAINMENU  := 2
STATE_LOADING   := 3
STATE_CHAT      := 4

currentState := STATE_NORMAL
lastDisconnectHandled := false

; -------------------------
; 获取窗口句柄
hwnd := WinExist("ahk_exe SCUM.exe")
if !hwnd {
    Log("❌ 未找到 SCUM.exe 窗口")
    return
}
Log("✅ 找到窗口 HWND=" hwnd)

; -------------------------
; 固定窗口大小 1280x720，左上角
winW := 1280
winH := 720
winX := 0
winY := 0
try {
    WinActivate(hwnd)
    WinWaitActive(hwnd,, 2)
    WinMove(winX, winY, winW, winH)
    Log("窗口已固定大小 1280x720 并移动到左上角")
} catch Error as e {
    Log("⚠️ WinMove失败: " e.Message)
    return
}

; 每秒检测一次
SetTimer(SCUM_Auto, 5000)
return

SCUM_Auto(*) {
    global gameTitle, disconnectOK, continueBtn, chatIcon
    global chatColorX, chatColorY, chatBlue
    global loadCheckX, loadCheckY, loadCheckColor
    global currentState, lastDisconnectHandled

    ; ========================
    ; 1️⃣ 掉线检测
    disconnectExist := ImageSearch(&bx, &by, 0, 0, winW, winH, "*65 " disconnectOK)
    ;Log("掉线OK按钮检测结果：" disconnectExist)
    ;Sleep 2000
    if disconnectExist{
        Log("检测到掉线OK按钮"  ", bx=" bx ", by=" by)
        Sleep 1000
        Click(winX + bx, winY + by)
        Log("💡 掉线 OK 已点击")
        Sleep 1000
        currentState := STATE_NORMAL
        return

    }

    ; ========================
    ; 2️⃣ 主菜单继续游戏
    continuebtnExist := ImageSearch(&bx, &by, 0, 0, winW, winH, "*65 " continueBtn)
    ;Log("主菜单继续游戏检测结果：" continuebtnExist)
    ;Sleep 2000
    if continuebtnExist {
        Log("检测到继续游戏按钮!"  ", bx=" bx ", by=" by)
        Sleep 1000
        Click(winX + bx, winY + by)
        Log("▶️ 继续游戏 已点击")
        Sleep 1000
        currentState := STATE_LOADING
        return
    }

    ; ========================
    ; 3️⃣ 游戏加载完成检测
    color:= PixelGetColor(loadCheckX, loadCheckY)
    gameLoaded := color != loadCheckColor
    if !gameLoaded {
        currentState := STATE_LOADING
        Log("PixelGetColor loadCheck: color=" color ", 游戏加载中=" gameLoaded)
        Sleep 1000
        return
    } else {
        currentState := STATE_CHAT
        Log("PixelGetColor loadCheck: color=" color ", 游戏加载完成=" gameLoaded)
        Sleep 1000
    }

    ; ========================
    ; 4️⃣ 聊天栏检测
    chatExists := ImageSearch(&bx, &by, 0, 0, winW, winH, "*85 " chatIcon)
    Log("聊天栏检测结果：" chatExists)
    ;Sleep 2000
    needSwitchChat := 0
    if !chatExists {
        Log("ImageSearch chatIcon not found!" "chatExists=" chatExists)
        ;Click(winX + 10, winY + 10)
        Sleep 1000
        Send "t"
        Log("💬 聊天栏不存在，已按 T")
        Sleep 1000
        needSwitchChat := 1
    }

    ; ========================
    ; 5️⃣ 聊天栏全球频道检测
    chatExists := ImageSearch(&bx, &by, 0, 0, winW, winH, "*60 " chatGlobal)
        if !chatExists {
        Log("ImageSearch chatIcon not found!" "chatExists=" chatExists)
        ;Click(winX + 10, winY + 10)
        Sleep 1000
        Send "{Tab}"
        Log("💬 非全球频道，已按 Tab")
        Sleep 1000
        needSwitchChat := 1
    }

    Log("✅ SCUM 自动检测完成")
}