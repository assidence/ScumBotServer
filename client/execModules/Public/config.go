package Public

import (
	"fmt"
	"gopkg.in/ini.v1"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// loadRules 从 ini 文件加载规则
func (lw *LogWatcher) loadRules() {
	files, _ := filepath.Glob(filepath.Join(lw.rulesDir, "*.ini"))
	var loaded []Rule
	for _, f := range files {
		stat, err := os.Stat(f)
		if err != nil {
			continue
		}
		if !stat.ModTime().After(lw.lastMod) {
			continue
		}
		cfg, err := ini.Load(f)
		if err != nil {
			continue
		}
		for _, section := range cfg.Sections() {
			if section.Name() == "DEFAULT" {
				continue
			}
			blockStart := section.Key("block_start").String()
			pattern := section.Key("pattern").String()
			if blockStart == "" || pattern == "" {
				continue
			}
			loaded = append(loaded, Rule{
				Name:       section.Name(),
				BlockStart: regexp.MustCompile(blockStart),
				Pattern:    regexp.MustCompile(pattern),
			})
		}
		if stat.ModTime().After(lw.lastMod) {
			lw.lastMod = stat.ModTime()
		}
	}
	lw.rules = loaded
	fmt.Println("[LogWatcher] 已加载规则:", len(lw.rules))
	//fmt.Println("Name:" + lw.rules[0].Name)
	//fmt.Println("BlockStart:")
	//fmt.Println(lw.rules[0].BlockStart)
	//fmt.Println("Pattern:")
	//fmt.Println(lw.rules[0].Pattern)
}

// autoReloadRules 热重载规则
func (lw *LogWatcher) autoReloadRules() {
	for {
		lw.loadRules()
		time.Sleep(5 * time.Second)
	}
}
