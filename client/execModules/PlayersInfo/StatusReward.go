package PlayersInfo

import (
	"ScumBotServer/client/execModules/Public"
	"bufio"
	"fmt"
	"gopkg.in/ini.v1"
	"os"
	"sync"
)

// ------------------ 配置与结构体 ------------------

type RewardConfig struct {
	Require  int      // 奖励命令文件路径
	Commands []string // 奖励命令列表
}

type RewardManager struct {
	configs  map[string]*RewardConfig
	counts   map[string]map[string]int // map[奖励名称]map[steamID]触发次数
	debug    bool
	chatChan chan string
	mu       sync.Mutex
}

// ------------------ 构造函数 ------------------

func NewRewardManager(iniPath string, chatChan chan string, enableDebug bool) (*RewardManager, error) {
	cfgFile, err := ini.Load(iniPath)
	if err != nil {
		return nil, err
	}

	configs := make(map[string]*RewardConfig)
	for _, section := range cfgFile.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}

		require, err := section.Key("Require").Int()
		if err != nil {
			return nil, fmt.Errorf("奖励 %s 配置 Require 错误: %v", section.Name(), err)
		}

		cmdFile := section.Key("RewardCommand").String()
		commands, err := readCommandsFromFile(cmdFile)
		if err != nil {
			return nil, fmt.Errorf("奖励 %s 读取命令文件失败: %v", section.Name(), err)
		}

		configs[section.Name()] = &RewardConfig{
			Require:  require,
			Commands: commands,
		}
	}

	rm := &RewardManager{
		configs:  configs,
		counts:   make(map[string]map[string]int),
		debug:    enableDebug,
		chatChan: chatChan,
	}

	rm.Debugf("初始化完成，共 %d 种奖励", len(configs))
	return rm, nil
}

// ------------------ 文件读取 ------------------

func readCommandsFromFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var commands []string
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			commands = append(commands, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return commands, nil
}

// ------------------ 调试输出 ------------------

func (rm *RewardManager) Debugf(format string, a ...interface{}) {
	if rm.debug {
		fmt.Printf("[StatusReward-Debug] "+format+"\n", a...)
	}
}

// ------------------ 定时器 ------------------
func (rm *RewardManager) Timer() {}

// ------------------ 触发奖励 ------------------

func (rm *RewardManager) TriggerRewards(trigger map[string][]string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for rewardName, steamIDs := range trigger {
		config, ok := rm.configs[rewardName]
		if !ok {
			rm.Debugf("未找到奖励配置: %s", rewardName)
			continue
		}

		if _, exists := rm.counts[rewardName]; !exists {
			rm.counts[rewardName] = make(map[string]int)
		}

		for _, sid := range steamIDs {
			rm.counts[rewardName][sid]++
			rm.Debugf("奖励 %s，玩家 %s 当前次数 %d/%d", rewardName, sid, rm.counts[rewardName][sid], config.Require)

			if rm.counts[rewardName][sid] >= config.Require {
				rm.Debugf("玩家 %s 达到奖励 %s 条件，执行奖励", sid, rewardName)
				rm.RewardAction(sid, rewardName)
				rm.counts[rewardName][sid] = 0
			}
		}
	}
}

func (rm *RewardManager) RewardAction(steamID string, rewardName string) {
	commands := rm.configs[rewardName].Commands
	for _, command := range commands {
		cfglines := Public.CommandSelecterInterface.Selecter(steamID, command)
		for _, line := range cfglines {
			rm.chatChan <- line
			fmt.Println("[StatusReward-Module]:" + line)
		}
	}
}
