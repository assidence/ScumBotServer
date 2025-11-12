package DBwatcher

import (
	"database/sql"
)

// QueryEquippedItemsBySteamIDs 查询指定 SteamIDs 的玩家装备信息。
// 返回 map[steamID][]装备名称
func QueryEquippedItemsBySteamIDs(db *sql.DB, steamIDs []string) (map[string][]string, error) {
	result := make(map[string][]string)

	for _, steamID := range steamIDs {
		result[steamID] = []string{} // 预先创建空列表
		//fmt.Printf("\n--- 开始处理 SteamID: %s ---\n", steamID)

		var userProfileID, prisonerID, entityID int

		// Step 1: 查询 user_profile.id
		err := db.QueryRow(`SELECT id FROM user_profile WHERE user_id = ?`, steamID).Scan(&userProfileID)
		if err != nil {
			//fmt.Printf("[WARN] 找不到 user_profile.id (steamID=%s): %v\n", steamID, err)
			continue
		}
		//fmt.Printf("[INFO] user_profile.id = %d\n", userProfileID)

		// Step 2: 查询 prisoner.id
		err = db.QueryRow(`SELECT id FROM prisoner WHERE user_profile_id = ?`, userProfileID).Scan(&prisonerID)
		if err != nil {
			//fmt.Printf("[WARN] 找不到 prisoner.id (user_profile_id=%d): %v\n", userProfileID, err)
			continue
		}
		//fmt.Printf("[INFO] prisoner.id = %d\n", prisonerID)

		// Step 3: 查询 prisoner_entity.entity_id
		err = db.QueryRow(`SELECT entity_id FROM prisoner_entity WHERE prisoner_id = ?`, prisonerID).Scan(&entityID)
		if err != nil {
			//fmt.Printf("[WARN] 找不到 prisoner_entity.entity_id (prisoner_id=%d): %v\n", prisonerID, err)
			continue
		}
		//fmt.Printf("[INFO] prisoner_entity.entity_id = %d\n", entityID)

		// Step 4: 查询所有装备 item_entity_id
		itemRows, err := db.Query(`SELECT item_entity_id FROM prisoner_inventory_equipped_item WHERE prisoner_entity_id = ?`, entityID)
		if err != nil {
			//fmt.Printf("[ERROR] 查询 prisoner_inventory_equipped_item 出错: %v\n", err)
			continue
		}

		itemIDs := []int{}
		for itemRows.Next() {
			var itemID int
			if err = itemRows.Scan(&itemID); err == nil {
				itemIDs = append(itemIDs, itemID)
			}
		}
		itemRows.Close()

		if len(itemIDs) == 0 {
			//fmt.Printf("[INFO] 玩家 %s 没有任何装备。\n", steamID)
			continue
		}

		//fmt.Printf("[INFO] 找到 %d 个装备 item_entity_id: %v\n", len(itemIDs), itemIDs)

		// Step 5: 根据 item_entity_id 查询 entity.class
		for _, itemID := range itemIDs {
			var itemClass string
			err = db.QueryRow(`SELECT class FROM entity WHERE id = ?`, itemID).Scan(&itemClass)
			if err == nil {
				//fmt.Printf("[WARN] 找不到 entity.class (id=%d): %v\n", itemID, err)
				continue
			}
			result[steamID] = append(result[steamID], itemClass)
			//fmt.Printf("[OK] 玩家 %s 装备物品: %s\n", steamID, itemClass)
		}
	}

	//fmt.Println("\n查询完成 ✅")
	return result, nil
}
