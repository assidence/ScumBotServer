package Public

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Player 玩家信息
type Player struct {
	SteamID        string
	Name           string
	Fame           string
	AccountBalance string
	GoldBalance    string
	LocationX      string
	LocationY      string
	LocationZ      string
	Prefix         string
}

// Vehicle 载具信息
/*
type Vehicle struct {
	VehicleType string
	VehicleID   string
}

*/

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

	lastOffset int64
	mu         sync.Mutex
	Players    map[string]Player
	Vehicles   map[string][]string
}

// NewLogWatcher 创建实例
func NewLogWatcher(filePath string, interval time.Duration, rulesDir string) *LogWatcher {
	lw := &LogWatcher{
		FilePath:   filePath,
		Interval:   interval,
		rulesDir:   rulesDir,
		Players:    make(map[string]Player),
		Vehicles:   make(map[string][]string),
		lastOffset: 0,
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
				if len(match) == 9 { //第0项是完整匹配 后面是捕获组
					player := Player{
						Name:           match[1],
						SteamID:        match[2],
						Fame:           match[3],
						AccountBalance: match[4],
						GoldBalance:    match[5],
						LocationX:      match[6],
						LocationY:      match[7],
						LocationZ:      match[8],
					}
					tempPlayers[player.SteamID] = player
					//fmt.Println("[LogWatcher] 捕获玩家：", match[1])
				}
				if len(match) == 3 {
					lw.Vehicles[match[1]] = append(lw.Vehicles[match[1]], match[2])
					fmt.Println("[LogWatcher] 捕获载具生成：", match[1], match[2])
				}
			}
			lw.mu.Lock()
			lw.Players = tempPlayers
			lw.mu.Unlock()
		}
	}
}

// GetPlayers 获取当前玩家列表
func (lw *LogWatcher) GetPlayers() map[string]Player {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	copy := make(map[string]Player, len(lw.Players))
	for k, v := range lw.Players {
		copy[k] = v
	}
	return copy
}

func RunLogWatcher(initChan chan struct{}) {
	// 日志文件路径
	roamingPath := os.Getenv("AppData")
	logFile := strings.Replace(roamingPath, `AppData\Roaming`, `AppData\Local\SCUM\Saved\Logs\SCUM.log`, 1)

	// 规则所在文件夹
	rulesDir := "./ini/LogWatcher/"

	// 创建 LogWatcher
	GlobalLogWatcher = NewLogWatcher(logFile, 1*time.Second, rulesDir)

	// 开始监控日志
	GlobalLogWatcher.Start()

	close(initChan)
}
