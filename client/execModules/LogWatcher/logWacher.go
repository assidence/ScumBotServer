package LogWatcher

import (
	"ScumBotServer/client/execModules/Public"
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Player = Public.Player

// Rule 单条匹配规则
type Rule struct {
	Name       string
	BlockStart *regexp.Regexp
	Pattern    *regexp.Regexp
}

// LogWatcher 日志监控模块
type LogWatcher struct {
	FilePath string
	Interval time.Duration

	rulesDir string
	rules    []Rule
	lastMod  time.Time

	lastOffset       int64
	mu               sync.Mutex
	Players          map[string]Player
	PlayersResetTime time.Duration
	Vehicles         map[string][]string
}

// NewLogWatcher 创建实例
func NewLogWatcher(filePath string, interval time.Duration, rulesDir string) *LogWatcher {
	lw := &LogWatcher{
		FilePath:         filePath,
		Interval:         interval,
		rulesDir:         rulesDir,
		Players:          make(map[string]Player),
		PlayersResetTime: 60 * time.Second,
		Vehicles:         make(map[string][]string),
		lastOffset:       0,
	}
	lw.loadRules()
	//go lw.autoReloadRules()
	return lw
}

// Start 开始监控日志
func (lw *LogWatcher) Start() {
	go func() {
		for {
			file, err := os.Open(lw.FilePath)
			if err != nil {
				fmt.Println("[LogWatcher] 无法打开日志:", err)
				time.Sleep(lw.Interval)
				continue
			}
			// 如果第一次启动，直接跳到文件末尾
			if lw.lastOffset == 0 {
				if stat, err := file.Stat(); err == nil {
					lw.lastOffset = stat.Size()
					fmt.Printf("[LogWatcher] 启动时跳过历史日志，当前位置: %d 字节\n", lw.lastOffset)
				}
			}

			stat, _ := file.Stat()
			// 文件被清空或轮转
			if stat.Size() < lw.lastOffset {
				lw.lastOffset = 0
			}

			file.Seek(lw.lastOffset, 0)
			scanner := bufio.NewScanner(file)
			var buffer []string

			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				buffer = append(buffer, line)
				if strings.HasSuffix(line, "!") {
					lw.parseBlock(buffer)
					buffer = nil
				}
			}

			lw.lastOffset, _ = file.Seek(0, 1)
			file.Close()
			time.Sleep(lw.Interval)
			lw.mu.Lock()
			lw.PlayersResetTime -= lw.Interval
			if lw.PlayersResetTime <= 0 {
				tempPlayers := make(map[string]Player)
				lw.Players = tempPlayers
				lw.PlayersResetTime = 60 * time.Second
			}
			lw.mu.Unlock()
			lw.PhasePlayerToInterface()
			//fmt.Println("[LogWatcher]已更新读取")
		}
	}()
}

// parseBlock 解析玩家信息块
func (lw *LogWatcher) parseBlock(block []string) {
	text := strings.Join(block, "\n") // 保留换行
	//fmt.Println("--------")
	//fmt.Println(text)
	for _, r := range lw.rules {
		//fmt.Println("debug1")
		//fmt.Println(r.BlockStart.MatchString(text))
		//fmt.Println(r.Pattern.MatchString(text))
		if r.BlockStart.MatchString(text) && r.Pattern.MatchString(text) {
			//fmt.Println("debug2")
			// 找出文本里所有匹配的玩家
			matches := r.Pattern.FindAllStringSubmatch(text, -1)
			tempPlayers := make(map[string]Player)
			for _, match := range matches {
				//fmt.Println("debug3")
				if len(match) == 10 { //第0项是完整匹配 后面是捕获组
					player := Player{
						Name:           match[1],
						SteamID:        match[3],
						Fame:           match[4],
						AccountBalance: match[5],
						GoldBalance:    match[6],
						LocationX:      match[7],
						LocationY:      match[8],
						LocationZ:      match[9],
					}
					tempPlayers[player.SteamID] = player
					lw.mu.Lock()
					lw.Players = tempPlayers
					lw.PlayersResetTime = 60 * time.Second
					lw.mu.Unlock()
					lw.PhasePlayerToInterface()
					//fmt.Println("[LogWatcher] 捕获玩家：", match[1])
				}
				if len(match) == 3 {
					lw.Vehicles[match[1]] = append(lw.Vehicles[match[1]], match[2])
					lw.PhaseVehicleToInterface()
					//fmt.Println("[LogWatcher] 捕获载具生成：", match[1], match[2])
				}
			}
		}
	}
}

// GetPlayers 获取当前玩家列表
func (lw *LogWatcher) GetPlayers() map[string]Player {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	c := make(map[string]Player, len(lw.Players))
	for k, v := range lw.Players {
		c[k] = v
	}
	return c
}

func (lw *LogWatcher) PhasePlayerToInterface() {
	Public.LogWatcherInterface.Mu.Lock()
	defer Public.LogWatcherInterface.Mu.Unlock()
	Public.LogWatcherInterface.Players = lw.Players
}

func (lw *LogWatcher) PhaseVehicleToInterface() {
	Public.LogWatcherInterface.Mu.Lock()
	defer Public.LogWatcherInterface.Mu.Unlock()
	Public.LogWatcherInterface.Vehicles = lw.Vehicles
}

func RunLogWatcher(initChan chan struct{}) {
	// 日志文件路径
	roamingPath := os.Getenv("AppData")
	logFile := strings.Replace(roamingPath, `AppData\Roaming`, `AppData\Local\SCUM\Saved\Logs\SCUM.log`, 1)

	// 规则所在文件夹
	rulesDir := "./ini/LogWatcher/"

	// 创建 LogWatcher
	lw := NewLogWatcher(logFile, 1*time.Second, rulesDir)

	// 开始监控日志
	lw.Start()
	Public.LogWatcherInterface.GetPlayers = lw.GetPlayers

	close(initChan)
}
