@echo off
chcp 65001 > nil
:START
cls
echo 启动 SCUM 服务器监控脚本...


start "" /b "D:\SteamLibrary\steamapps\common\SCUM Server\SCUM\Binaries\Win64\SCUMServer.exe" -log


echo 等待 180 秒以启动 server.exe...
timeout /t 180 /nobreak >nul


start "" /b "F:\Project\Goland\ScumBotServer\server\server.exe" 20500 "D:\SteamLibrary\steamapps\common\SCUM Server\SCUM\Saved\SaveFiles\Logs"

:WaitLoop

tasklist /fi "imagename eq SCUMServer.exe" | find /i "SCUMServer.exe" >nul
if errorlevel 1 (
    taskkill /im server.exe /f
    echo 所有程序已退出，等待 5 秒后重启...
    timeout /t 5 /nobreak >nul
    goto START
)

timeout /t 1 /nobreak >nul
goto WaitLoop
