package Public

import "sync"

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

type LogWatcherInterfaceStruct struct {
	Mu         sync.Mutex
	Players    map[string]Player
	Vehicles   map[string][]string
	GetPlayers func() map[string]Player
}

var LogWatcherInterface *LogWatcherInterfaceStruct

func init() {
	LogWatcherInterface = &LogWatcherInterfaceStruct{
		Players:  make(map[string]Player),
		Vehicles: make(map[string][]string),
	}
}
