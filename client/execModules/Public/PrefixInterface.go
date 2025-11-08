package Public

// TitleCommandType 指令类型
type TitleCommandType string

const (
	CommandGrant  TitleCommandType = "@给予称号" // 授予称号
	CommandRemove TitleCommandType = "@移除称号" // 移除称号
	CommandSet    TitleCommandType = "@设置称号" // 设置当前称号
	CommandUnSet  TitleCommandType = "@隐藏称号" // 隐藏当前称号
	CommandQuery  TitleCommandType = "@称号"
	CommandUse    TitleCommandType = "@使用称号" //查询拥有的称号

)

// TitleCommand 外部模块发送过来的指令
type TitleCommand struct {
	UserID  string
	Command TitleCommandType
	Title   string
	Done    chan struct{}
}

type TitleInterfaceStruct struct {
	PrefixHasTitle         func(userID, title string) (bool, error)
	PrefixGetActiveTitle   func(userID string) (string, error)
	OnlinePlayerPrefixList map[string][]string
	CmdCh                  chan TitleCommand
}

var TitleInterface = &TitleInterfaceStruct{}

func init() {
	TitleInterface.CmdCh = make(chan TitleCommand, 10)
	TitleInterface.OnlinePlayerPrefixList = make(map[string][]string)
}
