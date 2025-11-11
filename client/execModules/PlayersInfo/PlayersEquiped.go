package PlayersInfo

import (
	"database/sql"
	"fmt"
	"gopkg.in/ini.v1"
	"strings"
)

// LoadClothesItems 按类别读取 ./db/itemsDB.db 中 Clothes 表的物品信息。
// 返回值: map[category][]item_name
func LoadClothesItems(dbPath string) (map[string][]string, error) {
	result := make(map[string][]string)

	// 1️⃣ 打开数据库（只读模式）
	db, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("无法打开数据库: %v", err)
	}
	defer db.Close()

	// 2️⃣ 查询 category 和 item_name
	rows, err := db.Query(`SELECT category, item_name FROM Clothes`)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}
	defer rows.Close()

	// 3️⃣ 遍历结果并按类别分组
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

	//fmt.Println("[DBwatcher] 成功读取物品数据:", len(result), "个类别")
	return result, nil
}

// -------------------- 配置读取 --------------------
var cfg *ini.File

// LoadEquipmentConfig 读取 ini 文件，返回 ini.File 对象
func LoadEquipmentConfig(iniPath string) (*ini.File, error) {
	var err error
	cfg, err = ini.Load(iniPath)
	if err != nil {
		return nil, fmt.Errorf("无法读取 ini 文件: %v", err)
	}
	return cfg, nil
}

// -------------------- 玩家装备评估 --------------------

// EvaluatePlayerEquipment 根据已加载的配置评估玩家装备
// equipped: map[steamID][]itemNames
// cfg: ini 文件对象（已加载）
// 返回 map[ruleName][]steamID
func EvaluatePlayerEquipment(equipped map[string][]string) map[string][]string {
	result := make(map[string][]string)

	// 遍历每个配置段（如 [naturism]）
	for _, section := range cfg.Sections() {
		sectionName := section.Name()
		if sectionName == "DEFAULT" {
			continue
		}

		// 从配置中读取各类条件
		caUnEquipts := parseList(section.Key("CaUnEquipt").String())
		caEquipts := parseList(section.Key("CaEquipts").String())
		unEquipts := parseList(section.Key("UnEquipts").String())
		equipts := parseList(section.Key("Equipts").String())

		var matchedPlayers []string

		// 遍历每个玩家
		for steamID, items := range equipped {
			if matchPlayer(items, caUnEquipts, caEquipts, unEquipts, equipts) {
				matchedPlayers = append(matchedPlayers, steamID)
			}
		}

		result[sectionName] = matchedPlayers
	}

	return result
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

// matchPlayer 判断单个玩家是否符合条件
func matchPlayer(items, caUnEquipts, caEquipts, unEquipts, equipts []string) bool {
	// 转为统一小写
	toLowerList := func(lst []string) []string {
		out := make([]string, len(lst))
		for i, v := range lst {
			out[i] = strings.ToLower(v)
		}
		return out
	}

	lowerItems := toLowerList(items)
	caUnEquipts = toLowerList(caUnEquipts)
	caEquipts = toLowerList(caEquipts)
	unEquipts = toLowerList(unEquipts)
	equipts = toLowerList(equipts)

	// 判断是否包含辅助函数
	containsAny := func(targets []string) bool {
		for _, item := range lowerItems {
			for _, t := range targets {
				if strings.Contains(item, t) {
					return true
				}
			}
		}
		return false
	}

	// 判断逻辑
	if len(caUnEquipts) > 0 && containsAny(caUnEquipts) {
		return false
	}
	if len(caEquipts) > 0 && !containsAny(caEquipts) {
		return false
	}
	if len(unEquipts) > 0 && containsAny(unEquipts) {
		return false
	}
	if len(equipts) > 0 && !containsAny(equipts) {
		return false
	}

	return true
}
