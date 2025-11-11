package PlayersInfo

import (
	"database/sql"
	"fmt"
	"gopkg.in/ini.v1"
	"log"
	"strings"
)

// Debug 输出管理
var PEDebugEnabled = false // 可以在模块初始化或运行时控制开关

func pedebug(format string, a ...interface{}) {
	if PEDebugEnabled {
		log.Printf("[PlayersInfo-DEBUG] "+format, a...)
	}
}

// -------------------- 数据库读取 --------------------

// LoadClothesItems 按类别读取 ./db/itemsDB.db 中 Clothes 表的物品信息。
// 返回值: map[category][]item_name
func LoadClothesItems(dbPath string) (map[string][]string, error) {
	result := make(map[string][]string)

	db, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("无法打开数据库: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT category, item_name FROM Clothes`)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var category, itemName string
		if err := rows.Scan(&category, &itemName); err != nil {
			return nil, fmt.Errorf("解析行失败: %v", err)
		}
		result[category] = append(result[category], itemName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取结果时出错: %v", err)
	}

	return result, nil
}

// -------------------- INI 配置读取 --------------------

// EquipmentConfig 保存已解析的 INI 配置
type EquipmentConfig struct {
	Sections []*ini.Section
}

// LoadEquipmentConfig 读取 ini 文件并返回已解析对象
func LoadEquipmentConfig(iniPath string) (*EquipmentConfig, error) {
	cfg, err := ini.Load(iniPath)
	if err != nil {
		return nil, fmt.Errorf("无法读取 ini 文件: %v", err)
	}

	sections := []*ini.Section{}
	for _, s := range cfg.Sections() {
		if s.Name() == "DEFAULT" {
			continue
		}
		sections = append(sections, s)
	}

	return &EquipmentConfig{Sections: sections}, nil
}

// -------------------- 玩家装备评估 --------------------

// EvaluatePlayerEquipment 根据已加载的配置评估玩家装备
// equipped: map[steamID]interface{}，每个 interface{} 可以是 []string 或 []interface{}
// 返回 map[ruleName][]steamID
func EvaluatePlayerEquipment(equipped map[string]interface{}) map[string][]string {
	result := make(map[string][]string)

	if eqiupCfg == nil || len(eqiupCfg.Sections) == 0 {
		pedebug("[EvaluatePlayerEquipment] 配置为空或未加载")
		return result
	}

	for _, section := range eqiupCfg.Sections {
		if section == nil {
			continue
		}
		sectionName := section.Name()
		pedebug("----- 处理配置段: %s -----", sectionName)

		// 安全读取每个字段
		caUnEquipts := parseList(section.Key("CaUnEquipt").String())
		caEquipts := parseList(section.Key("CaEquipts").String())
		unEquipts := parseList(section.Key("UnEquipts").String())
		equipts := parseList(section.Key("Equipts").String())

		pedebug("配置段条件 - CaUnEquipts: %v", caUnEquipts)
		pedebug("配置段条件 - CaEquipts: %v", caEquipts)
		pedebug("配置段条件 - UnEquipts: %v", unEquipts)
		pedebug("配置段条件 - Equipts: %v", equipts)

		var matchedPlayers []string
		for steamID, itemsInterface := range equipped {
			pedebug("正在评估玩家: %s", steamID)
			itemsList := toStringSlice(itemsInterface)
			pedebug("玩家物品列表: %v", itemsList)

			if matchPlayer(itemsList, caUnEquipts, caEquipts, unEquipts, equipts) {
				pedebug("玩家符合条件，加入结果: %s", steamID)
				matchedPlayers = append(matchedPlayers, steamID)
			} else {
				pedebug("玩家不符合条件: %s", steamID)
			}
		}

		result[sectionName] = matchedPlayers
		pedebug("配置段处理完成: %s 符合条件玩家: %v", sectionName, matchedPlayers)
	}

	return result
}

// -------------------- 辅助函数 --------------------

// 将 interface{} 转成 []string
func toStringSlice(itemsInterface interface{}) []string {
	strs := []string{}
	if itemsInterface == nil {
		return strs
	}

	switch v := itemsInterface.(type) {
	case []string:
		return v
	case []interface{}:
		for _, i := range v {
			if s, ok := i.(string); ok {
				strs = append(strs, s)
			}
		}
	}
	return strs
}

// parseList 将逗号分隔的字符串转为 []string
func parseList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// matchPlayer 判断单个玩家是否符合条件，并打印 debug 信息
func matchPlayer(items, caUnEquipts, caEquipts, unEquipts, equipts []string) bool {
	pedebug("----- matchPlayer 调试开始 -----")
	pedebug("原始玩家物品: %v", items)
	pedebug("CaUnEquipts: %v", caUnEquipts)
	pedebug("CaEquipts: %v", caEquipts)
	pedebug("UnEquipts: %v", unEquipts)
	pedebug("Equipts: %v", equipts)

	if len(caUnEquipts) > 0 && containsAnyCA(caUnEquipts, items) {
		pedebug("不符合条件: 玩家装备包含 CaUnEquipts 中物品")
		return false
	}
	if len(caEquipts) > 0 && !containsAnyCA(caEquipts, items) {
		pedebug("不符合条件: 玩家装备不包含任何 CaEquipts 中物品")
		return false
	}
	if len(unEquipts) > 0 && containsAny(unEquipts, items) {
		pedebug("不符合条件: 玩家装备包含 UnEquipts 中物品")
		return false
	}
	if len(equipts) > 0 && !containsAny(equipts, items) {
		pedebug("不符合条件: 玩家装备不包含任何 Equipts 中物品")
		return false
	}

	pedebug("玩家符合条件 ✅")
	pedebug("----- matchPlayer 调试结束 -----")
	return true
}

// 精确匹配列表中的物品（任意匹配）
// 只要 items 中任意一个元素与 targets 中的任意一个目标匹配（加 "_ES" 后），即返回 true
func containsAny(targets []string, items []string) bool {
	pedebug("[containsAny] 开始匹配检查: items=%v, targets=%v", items, targets)

	for _, item := range items {
		for _, t := range targets {
			tWithSuffix := t + "_ES"
			//pedebug("[containsAny] 检查 item=%s 与 target=%s", item, tWithSuffix)

			if item == tWithSuffix {
				pedebug("[containsAny] ✅ 匹配成功: %s == %s", item, tWithSuffix)
				return true
			}
		}
	}

	pedebug("[containsAny] ❌ 未发现匹配项 (items=%v, targets=%v)", items, targets)
	return false
}

// 根据物品类别判断（任意匹配）
// 只要 items 中任意一个元素属于指定类别（targets）下的任意物品，即返回 true
func containsAnyCA(targets []string, items []string) bool {
	pedebug("[containsAnyCA] 开始类别匹配检查: items=%v, categories=%v", items, targets)

	for _, item := range items {
		for _, t := range targets {
			caList, exists := itemsDB[t]
			if !exists {
				pedebug("[containsAnyCA] ⚠️ category=%s 不存在于 itemsDB", t)
				continue
			}

			for _, caItem := range caList {
				caItemWithSuffix := caItem + "_ES"
				//pedebug("[containsAnyCA] 检查 item=%s 与 caItem=%s (类别=%s)", item, caItemWithSuffix, t)

				if item == caItemWithSuffix {
					pedebug("[containsAnyCA] ✅ 匹配成功: %s 属于类别 %s", item, t)
					return true
				}
			}
		}
	}

	pedebug("[containsAnyCA] ❌ 未发现匹配项 (items=%v, categories=%v)", items, targets)
	return false
}
