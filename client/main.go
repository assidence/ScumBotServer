package main

import (
	"ScumBotServer/client/execModules"
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
)

func startAutoReConnectEXE() *os.Process {
	// 获取当前 Go 程序的工作目录
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("获取工作目录失败:", err)
		return nil
	}

	// 基于工作目录组合 AHK exe 的路径
	ahkPath := filepath.Join(wd, "AHK", "AutoReConnect.exe")
	exeDir := filepath.Dir(ahkPath)

	cmd := exec.Command(ahkPath)
	cmd.Dir = exeDir

	// 可选：隐藏窗口（后台运行）
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	// 启动 AHK exe
	err = cmd.Start()
	if err != nil {
		fmt.Println("启动 AHK exe 出错:", err)
		return nil
	}

	fmt.Println("AHK exe 已启动，PID:", cmd.Process.Pid)
	return cmd.Process
}

func main() {
	// 捕获退出信号，确保 AHK 被关闭
	var ahkProcess *os.Process
	defer func() {
		if ahkProcess != nil {
			fmt.Println("程序退出，终止 AHK exe...")
			ahkProcess.Kill()
		}
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		fmt.Println("收到退出信号，程序结束")
		if ahkProcess != nil {
			ahkProcess.Kill()
		}
		os.Exit(0)
	}()

	address := ""
	reader := bufio.NewScanner(os.Stdin)
	fmt.Print("请输入ScumBot-服务端的地址: ")
	reader.Scan()
	address = reader.Text()
	if address == "" {
		fmt.Println("用户直接按了回车，使用默认值")
		address = "0.0.0.0:20500"
	}

	var execCommand = make(chan map[string]interface{})
	var sendChannel = make(chan []byte)
	re := regexp.MustCompile(`\{[^}]*\}`)
	go commandExecuter(execCommand, sendChannel)
	go execModules.FocusWindows("SCUM  ")

	// 启动 AHK exe
	ahkProcess = startAutoReConnectEXE()
	if ahkProcess == nil {
		fmt.Println("⚠️ AHK exe 启动失败，将继续运行主程序")
	}

	for {
		NetworkSignal := make(chan struct{})
		exitChan := make(chan struct{})
		conn := HttpClient(address)
		go commandReader(re, conn, execCommand, NetworkSignal, exitChan)
		go commandSender(conn, sendChannel, exitChan)
		<-NetworkSignal
		fmt.Println("[WatchDog] 检测到掉线，关闭当前连接并尝试重连")
		close(exitChan)
		conn.Close()
	}
}
