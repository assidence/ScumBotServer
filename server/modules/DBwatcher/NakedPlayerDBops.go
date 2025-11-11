package DBwatcher

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

// OpenDBRO 只读打开数据库
func OpenDBRO(dbPath string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?mode=ro&_journal_mode=WAL&_busy_timeout=2000", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		// 如果 immutable 不被支持，尝试普通只读模式
		fmt.Println("[DBwatcher-Warnning] immutable 不被支持，回退为 mode=ro ", err)
		db, err = sql.Open("sqlite3", "file:"+dbPath+"?mode=ro")
		if err != nil {
			return nil, err
		}
	}
	return db, nil
}

// QueryColumn 执行 SQL 并返回第一列所有值
func QueryColumn(db *sql.DB, query string, args ...interface{}) []string {
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Println("QueryColumn 查询失败:", err)
		return nil
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var val string
		if err := rows.Scan(&val); err != nil {
			log.Println("Scan 错误:", err)
			continue
		}
		results = append(results, val)
	}
	return results
}

// Unique 去重
func Unique(list []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range list {
		if !seen[v] {
			result = append(result, v)
			seen[v] = true
		}
	}
	return result
}

// GetPrisonerIDsByUserIDs 根据玩家 user_id 列表查询对应 prisoner_id
func GetPrisonerIDsByUserIDs(db *sql.DB, userIDs []string) []string {
	var prisonerIDs []string
	for _, uid := range userIDs {
		rows, err := db.Query(`SELECT prisoner_id FROM user_profile WHERE user_id = ?;`, uid)
		if err != nil {
			log.Println("查询 prisoner_id 失败:", err)
			continue
		}

		for rows.Next() {
			var pid string
			rows.Scan(&pid)
			prisonerIDs = append(prisonerIDs, pid)
		}
		rows.Close()
	}
	return Unique(prisonerIDs)
}

// GetNakedPlayers 根据玩家 user_id 列表查询裸体玩家
func GetNakedPlayers(db *sql.DB, userIDs []string) []string {
	if len(userIDs) == 0 {
		return nil
	}

	var nakedUsers []string

	for _, steamID := range userIDs {
		//fmt.Println("[DBwatcher] 检查 SteamID:", steamID)

		// 1️⃣ 查 user_profile.id
		userProfileIDs := QueryColumn(db, `SELECT id FROM main.user_profile WHERE user_id = ?;`, steamID)
		//fmt.Println("[DBwatcher] user_profile.id:", userProfileIDs)
		if len(userProfileIDs) == 0 {
			continue
		}

		// 2️⃣ 查 prisoner.id
		var prisonerIDs []string
		for _, upID := range userProfileIDs {
			pIDs := QueryColumn(db, `SELECT id FROM main.prisoner WHERE user_profile_id = ?;`, upID)
			//fmt.Println("[DBwatcher] prisoner.id for user_profile_id", upID, ":", pIDs)
			prisonerIDs = append(prisonerIDs, pIDs...)
		}
		if len(prisonerIDs) == 0 {
			continue
		}

		// 3️⃣ 查 prisoner_entity.entity_id
		var entityIDs []string
		for _, pid := range prisonerIDs {
			eIDs := QueryColumn(db, `SELECT entity_id FROM main.prisoner_entity WHERE prisoner_id = ?;`, pid)
			//fmt.Println("[DBwatcher] prisoner_entity.entity_id for prisoner_id", pid, ":", eIDs)
			entityIDs = append(entityIDs, eIDs...)
		}
		if len(entityIDs) == 0 {
			continue
		}

		// 4️⃣ 查 prisoner_inventory_equipped_item，没有出现在这个表中的 entity_id 表示裸体
		isNaked := false
		for _, eid := range entityIDs {
			equipped := QueryColumn(db, `SELECT prisoner_entity_id FROM main.prisoner_inventory_equipped_item WHERE prisoner_entity_id = ?;`, eid)
			//fmt.Println("[DBwatcher] equipped items for entity_id", eid, ":", equipped)
			if len(equipped) == 0 {
				isNaked = true
				//fmt.Println("[DBwatcher] 发现裸体实体 entity_id:", eid)
				break
			}
		}

		if isNaked {
			nakedUsers = append(nakedUsers, steamID)
			//fmt.Println("[DBwatcher] SteamID", steamID, "判定为裸体")
		} else {
			//fmt.Println("[DBwatcher] SteamID", steamID, "有装备，不算裸体")
		}
	}

	return Unique(nakedUsers)
}
