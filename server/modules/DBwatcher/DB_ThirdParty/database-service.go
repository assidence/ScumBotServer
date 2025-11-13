package DB_ThirdParty

import (
	"log"
	"math"
	"sync"
	"time"
)

// -----------------------------------------------------------
// 模块结构定义
// -----------------------------------------------------------

// PlayerStatsCache 缓存玩家统计信息
type PlayerStatsCache struct {
	TotalPlayers  int
	OnlinePlayers int
	ActiveSquads  int
	LastUpdate    time.Time
}

// GameWorldCache 缓存游戏世界信息
type GameWorldCache struct {
	GameTime    string
	Temperature string
	LastUpdate  time.Time
}

// Config 模块配置
type Config struct {
	CacheIntervalSeconds int
}

// Commands 模拟数据库指令引用
type Commands struct {
	GetTotalPlayerCount  func() (int, error)
	GetOnlinePlayerCount func() (int, error)
	GetActiveSquadCount  func() (int, error)
	GetGameTimeData      func() (string, error)
	GetWeatherData       func() (string, error)
}

// DatabaseService 模块主结构
type DatabaseService struct {
	cache struct {
		PlayerStats PlayerStatsCache
		GameWorld   GameWorldCache
	}
	config      Config
	commands    Commands
	initialized bool
	mu          sync.Mutex
}

// -----------------------------------------------------------
// 模块实例
// -----------------------------------------------------------

var globalService *DatabaseService
var once sync.Once

// GetService 获取单例实例
func GetService() *DatabaseService {
	once.Do(func() {
		globalService = &DatabaseService{
			config: Config{CacheIntervalSeconds: 60},
		}
	})
	return globalService
}

// -----------------------------------------------------------
// 初始化
// -----------------------------------------------------------

// Initialize 初始化数据库服务
func (s *DatabaseService) Initialize(cmds Commands, cacheInterval int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.commands = cmds
	s.config.CacheIntervalSeconds = cacheInterval
	s.initialized = true

	log.Printf("[DatabaseService] Initialized (interval=%ds)", cacheInterval)

	s.loadInitialData()
}

// -----------------------------------------------------------
// 内部数据加载
// -----------------------------------------------------------

func (s *DatabaseService) loadInitialData() {
	now := time.Now()
	s.cache.PlayerStats.LastUpdate = now
	s.cache.GameWorld.LastUpdate = now

	log.Printf("[DatabaseService] Loading initial cache data...")

	total, _ := s.safeCallInt(s.commands.GetTotalPlayerCount)
	online, _ := s.safeCallInt(s.commands.GetOnlinePlayerCount)
	squads, _ := s.safeCallInt(s.commands.GetActiveSquadCount)
	gameTime, _ := s.safeCallString(s.commands.GetGameTimeData)
	weather, _ := s.safeCallString(s.commands.GetWeatherData)

	s.cache.PlayerStats.TotalPlayers = total
	s.cache.PlayerStats.OnlinePlayers = online
	s.cache.PlayerStats.ActiveSquads = squads
	s.cache.PlayerStats.LastUpdate = now

	s.cache.GameWorld.GameTime = gameTime
	s.cache.GameWorld.Temperature = weather
	s.cache.GameWorld.LastUpdate = now

	log.Printf("[DatabaseService] Initial cache loaded: Total=%d, Online=%d, Squads=%d",
		total, online, squads)
	log.Printf("[DatabaseService] GameWorld: Time=%s, Temp=%s", gameTime, weather)
}

// -----------------------------------------------------------
// 更新缓存
// -----------------------------------------------------------

func (s *DatabaseService) UpdateCacheIfNeeded() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		log.Println("[DatabaseService] Not initialized.")
		return
	}

	now := time.Now()
	interval := time.Duration(s.config.CacheIntervalSeconds) * time.Second

	playerAge := now.Sub(s.cache.PlayerStats.LastUpdate)
	worldAge := now.Sub(s.cache.GameWorld.LastUpdate)
	maxAge := math.Max(playerAge.Seconds(), worldAge.Seconds())

	if maxAge < interval.Seconds() {
		log.Printf("[DatabaseService] Cache is fresh (%.1fs old), skip update.", maxAge)
		return
	}

	log.Printf("[DatabaseService] Cache stale (%.1fs old), updating...", maxAge)

	total, _ := s.safeCallInt(s.commands.GetTotalPlayerCount)
	online, _ := s.safeCallInt(s.commands.GetOnlinePlayerCount)
	squads, _ := s.safeCallInt(s.commands.GetActiveSquadCount)
	gameTime, _ := s.safeCallString(s.commands.GetGameTimeData)
	weather, _ := s.safeCallString(s.commands.GetWeatherData)

	s.cache.PlayerStats.TotalPlayers = total
	s.cache.PlayerStats.OnlinePlayers = online
	s.cache.PlayerStats.ActiveSquads = squads
	s.cache.PlayerStats.LastUpdate = now

	s.cache.GameWorld.GameTime = gameTime
	s.cache.GameWorld.Temperature = weather
	s.cache.GameWorld.LastUpdate = now

	log.Printf("[DatabaseService] Updated cache: Total=%d Online=%d Squads=%d", total, online, squads)
	log.Printf("[DatabaseService] World: Time=%s Temp=%s", gameTime, weather)
}

// -----------------------------------------------------------
// 获取缓存数据
// -----------------------------------------------------------

func (s *DatabaseService) GetStats() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return map[string]interface{}{
			"TotalPlayers":  0,
			"OnlinePlayers": 0,
			"ActiveSquads":  0,
			"GameTime":      "N/A",
			"Temperature":   "N/A",
		}
	}

	return map[string]interface{}{
		"TotalPlayers":     s.cache.PlayerStats.TotalPlayers,
		"OnlinePlayers":    s.cache.PlayerStats.OnlinePlayers,
		"ActiveSquads":     s.cache.PlayerStats.ActiveSquads,
		"GameTime":         s.cache.GameWorld.GameTime,
		"Temperature":      s.cache.GameWorld.Temperature,
		"LastPlayerUpdate": s.cache.PlayerStats.LastUpdate,
		"LastWorldUpdate":  s.cache.GameWorld.LastUpdate,
	}
}

// -----------------------------------------------------------
// 获取缓存状态信息
// -----------------------------------------------------------

func (s *DatabaseService) GetCacheInfo() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	playerAge := now.Sub(s.cache.PlayerStats.LastUpdate).Seconds()
	worldAge := now.Sub(s.cache.GameWorld.LastUpdate).Seconds()
	maxAge := math.Max(playerAge, worldAge)

	valid := maxAge < float64(s.config.CacheIntervalSeconds)
	estCalls := (3600 / s.config.CacheIntervalSeconds) * 5

	return map[string]interface{}{
		"CacheInterval":         s.config.CacheIntervalSeconds,
		"PlayerStatsAgeSeconds": playerAge,
		"GameWorldAgeSeconds":   worldAge,
		"CacheValid":            valid,
		"EstimatedCallsPerHour": estCalls,
	}
}

// -----------------------------------------------------------
// 安全调用工具函数
// -----------------------------------------------------------

func (s *DatabaseService) safeCallInt(fn func() (int, error)) (int, error) {
	if fn == nil {
		return 0, nil
	}
	val, err := fn()
	if err != nil {
		log.Printf("[DatabaseService] call error: %v", err)
		return 0, err
	}
	return val, nil
}

func (s *DatabaseService) safeCallString(fn func() (string, error)) (string, error) {
	if fn == nil {
		return "N/A", nil
	}
	val, err := fn()
	if err != nil {
		log.Printf("[DatabaseService] call error: %v", err)
		return "N/A", err
	}
	return val, nil
}
