module ScumBotServer

go 1.25.2

require (
	github.com/mattn/go-sqlite3 v1.14.32
	golang.org/x/text v0.30.0
	gopkg.in/ini.v1 v1.67.0
)

require github.com/stretchr/testify v1.11.1 // indirect

replace github.com/otiai10/gosseract/v2 => github.com/otiai10/gosseract/v2 v2.4.1
