package modules

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

func FindNewestChatLog(dirPath string) (string, time.Time, error) {
	pattern := `^chat_\d{14}\.log$`
	var newestFile string
	var newestTime time.Time

	entries, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return "", time.Time{}, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue //skip substance folder
		}

		//get files info
		modtime := entry.ModTime()
		if modtime.After(newestTime) {
			newestTime = modtime
			if success, _ := regexp.MatchString(pattern, entry.Name()); success {
				newestFile = filepath.Join(dirPath, entry.Name())
			}
		}
	}

	if newestFile == "" {
		return "", time.Time{}, os.ErrNotExist
	}

	return newestFile, newestTime, nil
}
