package Public

type CommandSelecterInterfaceStruct struct {
	Selecter func(steamID string, cfgCommand string) []string
}

var CommandSelecterInterface *CommandSelecterInterfaceStruct

func init() {
	CommandSelecterInterface = &CommandSelecterInterfaceStruct{}
}
