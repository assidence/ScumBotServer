package DBwatcher

import (
	"database/sql"
	"fmt"
	"strings"
)

// QueryEquippedItems 查询指定 SteamIDs 的玩家装备信息。
// 返回值：map[steamID][]装备名
func QueryEquippedItems(db *sql.DB, steamIDs []string) (map[string][]string, error) {
	result := make(map[string][]string)

	if len(steamIDs) == 0 {
		return result, nil
	}

	// 1️⃣ 构建动态占位符 (?, ?, ?, ...)
	placeholders := make([]string, len(steamIDs))
	args := make([]interface{}, len(steamIDs))
	for i, id := range steamIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	// 2️⃣ 构建查询语句
	query := fmt.Sprintf(`
		SELECT 
			user_profile.user_id AS steam_id,
			entity.class AS item_name
		FROM prisoner_inventory_equipped_item
		INNER JOIN prisoner_entity ON prisoner_inventory_equipped_item.prisoner_entity_id = prisoner_entity.entity_id
		INNER JOIN prisoner ON prisoner_entity.prisoner_id = prisoner.id
		INNER JOIN entity ON prisoner_inventory_equipped_item.item_entity_id = entity.id
		INNER JOIN user_profile ON user_profile.id = prisoner.user_profile_id
		WHERE user_profile.user_id IN (%s)
	`, strings.Join(placeholders, ","))

	// 3️⃣ 执行查询
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}
	defer rows.Close()

	// 4️⃣ 解析结果
	for rows.Next() {
		var steamID, itemName string
		if err := rows.Scan(&steamID, &itemName); err != nil {
			return nil, fmt.Errorf("解析结果出错: %v", err)
		}
		result[steamID] = append(result[steamID], itemName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果出错: %v", err)
	}

	return result, nil
}
