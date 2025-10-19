package modules

import (
	"ScumBotServer/server/modules/tail"
)

func ReadCommand(filePath string) *chan *Utf16tail.Line {
	t := Utf16tail.New(filePath)
	return &t.Lines
}
