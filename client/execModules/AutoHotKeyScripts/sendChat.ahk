; send_chat.ahk
; 参数1: 游戏进程名 (例: Game.exe)
; 参数2: 要发送的消息

ProcessName := %1%
MsgText := %2%

; 聚焦游戏窗口
IfWinExist, ahk_exe %ProcessName%
{
    WinActivate
    WinWaitActive, ahk_exe %ProcessName%, , 0.05  ; 等待50ms窗口激活
}
else
{
    MsgBox, 找不到进程: %ProcessName%
    ExitApp
}

; 打开聊天框
Send, {t}
Sleep, 50

; Ctrl+A 全选残留文本
Send, ^a
Sleep, 20

; 输入消息
Send, %MsgText%
Sleep, 20

; 回车发送
Send, {Enter}
Return