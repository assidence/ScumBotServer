package modules

import (
	Utf16tail "ScumBotServer/modules/tail"
)

func ReadCommand(filePath string) *chan *Utf16tail.Line {
	t := Utf16tail.New(filePath)
	return &t.Lines
}
