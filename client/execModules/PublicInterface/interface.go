package PublicInterface

import (
	"database/sql"
	"sync"
)

type TitleCommandType string

const (
	CommandGrant  TitleCommandType = "@给予称号"
	CommandRemove TitleCommandType = "@移除称号"
	CommandSet    TitleCommandType = "@设置称号"
	CommandUnSet  TitleCommandType = "@隐藏称号"
)

// TitleCommand 外部模块发送过来的指令
type TitleCommand struct {
	UserID  string
	Command TitleCommandType
	Title   string
	Done    chan struct{}
}

// TitleManager 核心管理器
type TitleManagerStructure struct {
	db     *sql.DB
	CmdCh  chan TitleCommand
	mu     sync.Mutex
	wg     sync.WaitGroup
	closed bool
}
