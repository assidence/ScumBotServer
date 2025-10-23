package modules

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

func FindNewestEconomyLog(dirPath string) (string, time.Time, error) {
	pattern := regexp.MustCompile(`^economy_\d{14}\.log$`)
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

		if pattern.MatchString(entry.Name()) {
			if entry.ModTime().After(newestTime) {
				newestTime = entry.ModTime()
				newestFile = filepath.Join(dirPath, entry.Name())
			}
		}
	}

	if newestFile == "" {
		return "", time.Time{}, os.ErrNotExist
	}

	return newestFile, newestTime, nil
}

func FindNewestChatLog(dirPath string) (string, time.Time, error) {
	pattern := regexp.MustCompile(`^chat_\d{14}\.log$`)
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

		if pattern.MatchString(entry.Name()) {
			if entry.ModTime().After(newestTime) {
				newestTime = entry.ModTime()
				newestFile = filepath.Join(dirPath, entry.Name())
			}
		}
	}

	if newestFile == "" {
		return "", time.Time{}, os.ErrNotExist
	}

	return newestFile, newestTime, nil
}

func FindNewestLoginLog(dirPath string) (string, time.Time, error) {
	// 匹配 login_20251006102826.log 这样的文件名
	pattern := regexp.MustCompile(`^login_\d{14}\.log$`)

	var newestFile string
	var newestTime time.Time

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", time.Time{}, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // 跳过子目录
		}

		if pattern.MatchString(entry.Name()) {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(newestTime) {
				newestTime = info.ModTime()
				newestFile = filepath.Join(dirPath, entry.Name())
			}
		}
	}

	if newestFile == "" {
		return "", time.Time{}, os.ErrNotExist
	}

	return newestFile, newestTime, nil
}
