#Requires AutoHotkey v2.0.18+ 64-bit

F12::ExitApp

Debug_Mode := true
LogFile := A_ScriptDir "\debug.log"

Log(msg, mode := "file") {
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

chatColorX := 365
chatColorY := 307
;chatBlue   := 0x122E34

loadCheckX := 90
loadCheckY := 145
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
SetTimer(SCUM_Auto, 1000)
return

SCUM_Auto(*) {
    global gameTitle, disconnectOK, continueBtn, chatIcon
    global chatColorX, chatColorY, chatBlue
    global loadCheckX, loadCheckY, loadCheckColor
    global currentState, lastDisconnectHandled

    ; ========================
    ; 1️⃣ 掉线检测
    disconnectExist := ImageSearch(&bx, &by, 0, 0, winW, winH, "*85 " disconnectOK)
    ;Log("掉线OK按钮检测结果：" disconnectExist)
    ;Sleep 2000
    if disconnectExist{
        Log("ImageSearch disconnectOK detected!"  ", bx=" bx ", by=" by)

        Click(winX + bx, winY + by)
        Log("💡 掉线 OK 已点击")
        Sleep 500
        currentState := STATE_NORMAL
        return

    }

    ; ========================
    ; 2️⃣ 主菜单继续游戏
    continuebtnExist := ImageSearch(&bx, &by, 0, 0, winW, winH, "*85 " continueBtn)
    ;Log("主菜单继续游戏检测结果：" continuebtnExist)
    ;Sleep 2000
    if continuebtnExist {
        Log("ImageSearch continueButton detected!"  ", bx=" bx ", by=" by)
        Sleep 500
        Click(winX + bx, winY + by)
        Log("▶️ 继续游戏 已点击")
        currentState := STATE_LOADING
        return
    }

    ; ========================
    ; 3️⃣ 游戏加载完成检测
    color:= PixelGetColor(loadCheckX, loadCheckY)
    gameLoaded := color != loadCheckColor
    ;Sleep 2000
    if !gameLoaded {
        currentState := STATE_LOADING
        Log("PixelGetColor loadCheck: color=" color ", gameLoaded=" gameLoaded)
        return
    } else {
        currentState := STATE_CHAT
        Log("PixelGetColor loadCheck: color=" color ", gameLoaded=" gameLoaded)
    }

    ; ========================
    ; 4️⃣ 聊天栏检测
    chatExists := ImageSearch(&bx, &by, 0, 0, winW, winH, "*85 " chatIcon)
    Log("聊天栏检测结果：" chatExists)
    ;Sleep 2000
    needSwitchChat := 0
    if !chatExists {
        Log("ImageSearch chatIcon not found!" "chatExists=" chatExists)
        Click(winX + 10, winY + 10)
        Sleep 500
        Send "t"
        Log("💬 聊天栏不存在，已按 T")
        needSwitchChat := 1
    }

    ; ========================
    ; 5️⃣ 聊天栏颜色检测
    ; 假设三个频道的像素坐标
    if needSwitchChat == 0{
        return
    }
    chatCoords := [[chatColorX, chatColorY], [chatColorX, chatColorY], [chatColorX, chatColorY]]

    maxBlue := -1
    targetIndex := 0

    Loop 3 {
        x := chatCoords[A_Index][1]
        y := chatCoords[A_Index][2]
        color := PixelGetColor(x, y, true) ; true 表示返回 0xRRGGBB

        r := (color >> 16) & 0xFF
        g := (color >> 8) & 0xFF
        b := color & 0xFF

        blueScore := b - ((r + g) / 2)

        if (blueScore > maxBlue) {
            maxBlue := blueScore
            targetIndex := A_Index
        }
        Log("当前循环" String(A_Index) "蓝色值：" String(maxBlue) "记录频道：" String(targetIndex))
        Send "{Tab}"
        Sleep 500
    }

    Log("最蓝的频道是第" targetIndex "个，蓝色值：" string(maxBlue))

    ; 自动切换到目标频道
    ; 假设当前频道从 1 开始，用 Tab 循环
    currentIndex := 1
    while (currentIndex != targetIndex) {
        Log("当前频道:" String(currentIndex) "目标频道:" String(targetIndex))
        Send "{Tab}"
        Sleep 500
        currentIndex := currentIndex + 1
    }
    needSwitchChat := 0
    Log("✅ SCUM 自动检测完成")
}