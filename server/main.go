package main

import (
	modules2 "ScumBotServer/server/modules"
	"ScumBotServer/server/modules/DBwatcher"
	IMServer2 "ScumBotServer/server/modules/IMServer"
	Utf16tail "ScumBotServer/server/modules/tail"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

func PlayerCommandSender(PlayerCommand *chan *Utf16tail.Line, commch chan string) {
	for line := range *PlayerCommand {
		commch <- line.Text
		//fmt.Printf("[Network] Broadcast:\n", line.Text)
	}
}

func LoginCommandSender(LoginCommand *chan *Utf16tail.Line, commch chan string) {
	for line := range *LoginCommand {
		commch <- line.Text
		//fmt.Printf("[Network] Broadcast:\n", line.Text)
	}
}

func EconomyCommandSender(EconomyCommand *chan *Utf16tail.Line, ecoch chan string) {
	for line := range *EconomyCommand {
		ecoch <- line.Text
		//fmt.Printf("[Network] Broadcast:\n", line.Text)
	}
}

func KillCommandSender(KillCommand *chan *Utf16tail.Line, killch chan string) {
	for line := range *KillCommand {
		killch <- line.Text
		//fmt.Printf("[Network] Broadcast:\n", line.Text)
	}
}

// 客户端信息分流器
func IncomeMessagesSelector(imcomech chan IMServer2.Message, DBWincomech chan map[string]interface{}) {
	for msg := range imcomech {
		Content := msg.Content
		//fmt.Println("Content:", Content)
		var result map[string]interface{}
		err := json.Unmarshal([]byte(Content), &result)
		if err != nil {
			fmt.Println("[IncomeMessageSelector] 解析失败:", err)
			continue
		}
		//fmt.Println("[IncomeMessageSelector] 解析结果：", result)
		if result["type"] == nil {
			//fmt.Println("[IncomeMessageSelector] 非服务端任务 已跳过：", result["type"])
			continue
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		switch result["type"].(string) {
		case "onlinePlayers":
			DBWincomech <- result

			//fmt.Println(now, "[IncomeMessageSelector] 识别为DBWatcher任务")
		default:
			fmt.Println(now, "[IncomeMessageSelector] 无法识别任务类型", result["type"])
		}
	}
}

func main() {
	//found newset ChatLog
	filePath, newsetTime, err := modules2.FindNewestChatLog(os.Args[2])
	if err != nil {
		panic(err)
	}
	fmt.Printf("[Success]Chat log founded!%s(%s)\n", filePath, newsetTime)
	var PlayerCommand = modules2.ReadCommand(filePath)
	fmt.Printf("[Success]Chat log Loaded!\n")

	//found newset LoginLog
	filePath, newsetTime, err = modules2.FindNewestLoginLog(os.Args[2])
	if err != nil {
		panic(err)
	}
	fmt.Printf("[Success]login log founded!%s(%s)\n", filePath, newsetTime)
	var LoginCommand = modules2.ReadCommand(filePath)
	fmt.Printf("[Success]login log Loaded!\n")

	//found newset EconomyLog
	filePath, newsetTime, err = modules2.FindNewestEconomyLog(os.Args[2])
	if err != nil {
		panic(err)
	}
	fmt.Printf("[Success]Economy log founded!%s(%s)\n", filePath, newsetTime)
	var EconomyCommand = modules2.ReadCommand(filePath)
	fmt.Printf("[Success]Economy log Loaded!\n")

	//found newset KillEventLog
	filePath, newsetTime, err = modules2.FindNewestKillEventLog(os.Args[2])
	if err != nil {
		panic(err)
	}
	fmt.Printf("[Success]Kill log founded!%s(%s)\n", filePath, newsetTime)
	var KillCommand = modules2.ReadCommand(filePath)
	fmt.Printf("[Success]Kill log Loaded!\n")

	addr := fmt.Sprintf("0.0.0.0:%s", os.Args[1])
	online := make(chan struct{})
	execch := make(chan string, 128)
	incomech := make(chan IMServer2.Message)
	go IMServer2.StartServer(addr, incomech, online)
	for {
		select {
		case <-online:
			break
		default:
			continue
		}
		break
	}
	go IMServer2.HttpClient(addr, execch)
	//玩家聊天框
	commch := make(chan string)
	reg := `'(\d+):([^']+)'\s+'.*?:\s*(@[^']+)'`
	go PlayerCommandSender(PlayerCommand, commch)
	go modules2.CommandHandler(reg, commch, execch)
	//玩家进入退出
	logch := make(chan string)
	reg = `^(\d{4}\.\d{2}\.\d{2}-\d{2}\.\d{2}\.\d{2}): '([\d\.]+) (\d+):([^()']+)\((\d+)\)' logged (in|out) at: X=([-\d\.]+) Y=([-\d\.]+) Z=([-\d\.]+)(?: \(as drone\))?$`
	go LoginCommandSender(LoginCommand, logch)
	go modules2.JoinLeaveHandler(reg, logch, execch)
	//玩家经济行为
	ecoch := make(chan string)
	//reg = `^(\d{4}\.\d{2}\.\d{2}-\d{2}\.\d{2}\.\d{2}): '([\d\.]+) (\d+):([^()']+)\((\d+)\)' logged (in|out) at: X=([-\d\.]+) Y=([-\d\.]+) Z=([-\d\.]+)(?: \(as drone\))?$`
	go EconomyCommandSender(EconomyCommand, ecoch)
	go modules2.EconomyHandler(ecoch, execch)
	//玩家击杀行为
	killch := make(chan string)
	reg = `Died: [^()]+\s\((\d+)\), Killer: [^()]+\s\((\d+)\) Weapon: ([^\[]+)\s\[(\w+)\]`
	go KillCommandSender(KillCommand, killch)
	go modules2.KillHandler(reg, killch, execch)

	//数据库监控
	DBWincomech := make(chan map[string]interface{})
	go DBwatcher.Start(execch, DBWincomech)

	go IncomeMessagesSelector(incomech, DBWincomech)
	select {}
}
