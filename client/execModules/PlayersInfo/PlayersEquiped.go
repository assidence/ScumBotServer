package PlayersInfo

import (
	"database/sql"
	"fmt"
	"gopkg.in/ini.v1"
	"strings"
)

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
// cfg: 已加载 EquipmentConfig 对象
// 返回 map[ruleName][]steamID
func EvaluatePlayerEquipment(equipped map[string]interface{}) map[string][]string {
	result := make(map[string][]string)

	if eqiupCfg == nil || len(eqiupCfg.Sections) == 0 {
		fmt.Println("[EvaluatePlayerEquipment] 配置为空或未加载")
		return result
	}

	for _, section := range eqiupCfg.Sections {
		if section == nil {
			continue
		}
		sectionName := section.Name()
		fmt.Println("----- 处理配置段:", sectionName, "-----")

		// 安全读取每个字段
		caUnEquipts := parseList(section.Key("CaUnEquipt").String())
		caEquipts := parseList(section.Key("CaEquipts").String())
		unEquipts := parseList(section.Key("UnEquipts").String())
		equipts := parseList(section.Key("Equipts").String())

		fmt.Println("配置段条件 - CaUnEquipts:", caUnEquipts)
		fmt.Println("配置段条件 - CaEquipts:", caEquipts)
		fmt.Println("配置段条件 - UnEquipts:", unEquipts)
		fmt.Println("配置段条件 - Equipts:", equipts)

		var matchedPlayers []string
		for steamID, itemsInterface := range equipped {
			fmt.Println("正在评估玩家:", steamID)
			itemsList := toStringSlice(itemsInterface)
			fmt.Println("玩家物品列表:", itemsList)

			if matchPlayer(itemsList, caUnEquipts, caEquipts, unEquipts, equipts) {
				fmt.Println("玩家符合条件，加入结果:", steamID)
				matchedPlayers = append(matchedPlayers, steamID)
			} else {
				fmt.Println("玩家不符合条件:", steamID)
			}
		}

		result[sectionName] = matchedPlayers
		fmt.Println("配置段处理完成:", sectionName, "符合条件玩家:", matchedPlayers)
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
	fmt.Println("----- matchPlayer 调试开始 -----")
	fmt.Println("原始玩家物品:", items)
	fmt.Println("CaUnEquipts:", caUnEquipts)
	fmt.Println("CaEquipts:", caEquipts)
	fmt.Println("UnEquipts:", unEquipts)
	fmt.Println("Equipts:", equipts)

	if len(caUnEquipts) > 0 && containsAnyCA(caUnEquipts, items) {
		fmt.Println("不符合条件: 玩家装备包含 CaUnEquipts 中物品")
		return false
	}
	if len(caEquipts) > 0 && !containsAnyCA(caEquipts, items) {
		fmt.Println("不符合条件: 玩家装备不包含任何 CaEquipts 中物品")
		return false
	}
	if len(unEquipts) > 0 && containsAny(unEquipts, items) {
		fmt.Println("不符合条件: 玩家装备包含 UnEquipts 中物品")
		return false
	}
	if len(equipts) > 0 && !containsAny(equipts, items) {
		fmt.Println("不符合条件: 玩家装备不包含任何 Equipts 中物品")
		return false
	}

	fmt.Println("玩家符合条件 ✅")
	fmt.Println("----- matchPlayer 调试结束 -----")
	return true
}

func containsAny(targets []string, items []string) bool {
	for _, item := range items {
		for _, t := range targets {
			if item == t {
				return true
			}
		}
	}
	return false
}

func containsAnyCA(targets []string, items []string) bool {
	for _, item := range items {
		for _, t := range targets {
			for _, caItem := range itemsDB[t] {
				fmt.Println(item, "-", caItem)
				if item == caItem {
					return true
				}
			}
		}
	}
	return false
}
