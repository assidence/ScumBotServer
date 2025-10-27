package DBwatcher

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

var execch chan string

func Start(execchAg chan string) {
	// 要监控的数据库路径
	dbPath := strings.Replace(os.Args[2], "Logs", "SCUM.db", 1)
	fmt.Println("dbPath:", dbPath)

	// 创建 watcher
	watcher, err := New(dbPath, 5*time.Second, onDatabaseUpdated)
	if err != nil {
		log.Fatal("创建监控器失败:", err)
	}
	execch = execchAg
	// 启动监控
	go watcher.Start()

	// 模拟程序持续运行
	select {}
}

// sequenceJson sequence dict to Json
func sequenceJson(execData *map[string]string) []byte {
	jsonByte, _ := json.Marshal(execData)
	return jsonByte
}

// onDatabaseUpdated 在数据库更新时触发
func onDatabaseUpdated(db *sql.DB) {
	var execData = map[string]string{
		"steamID":     "000000",
		"nickName":    "System",
		"command":     "0",
		"commandArgs": "0",
	}

	// 调用封装好的检测函数
	detectNaked(db)
	jsonByte := sequenceJson(&execData)
	execch <- string(jsonByte)
}

// detectNaked 执行未装备实体检测及用户查询
func detectNaked(db *sql.DB) {
	fmt.Println("数据库更新，开始检查未装备实体...")

	// 1️⃣ 取所有 entity_id
	allEntities, err := queryColumn(db, "SELECT entity_id FROM prisoner_entity;")
	if err != nil {
		log.Println("查询 prisoner_entity 失败:", err)
		return
	}

	// 2️⃣ 取所有已装备的 entity_id
	equippedEntities, err := queryColumn(db, "SELECT prisoner_entity_id FROM prisoner_inventory_equipped_item;")
	if err != nil {
		log.Println("查询 prisoner_inventory_equipped_item 失败:", err)
		return
	}

	// 3️⃣ 取未装备的 entity_id
	unEquippedEntities := difference(allEntities, equippedEntities)
	if len(unEquippedEntities) == 0 {
		fmt.Println("所有实体均已装备物品。")
		return
	}

	fmt.Printf("检测到 %d 个未装备实体，继续查询对应囚犯...\n", len(unEquippedEntities))

	// 4️⃣ 根据 entity_id 找 prisoner_id
	prisonerIDs := getPrisonerIDsByEntities(db, unEquippedEntities)

	// 5️⃣ 根据 prisoner_id 找 user_id
	userIDs := getUserIDsByPrisoners(db, prisonerIDs)

	// 6️⃣ 输出最终结果
	if len(userIDs) == 0 {
		fmt.Println("未找到对应的用户。")
		return
	}

	fmt.Println("以下用户存在未装备物品的实体：")
	for _, uid := range userIDs {
		fmt.Println(" - user_id:", uid)
	}
}

// queryColumn 执行 SQL 并返回第一列所有值
func queryColumn(db *sql.DB, query string, args ...interface{}) ([]string, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var val string
		if err := rows.Scan(&val); err != nil {
			return nil, err
		}
		results = append(results, val)
	}
	return results, nil
}

// difference 返回 listA 中有但 listB 中没有的值
func difference(listA, listB []string) []string {
	m := make(map[string]bool)
	for _, v := range listB {
		m[v] = true
	}
	var diff []string
	for _, v := range listA {
		if !m[v] {
			diff = append(diff, v)
		}
	}
	return diff
}

// getPrisonerIDsByEntities 根据 entity_id 查询对应的 prisoner_id
func getPrisonerIDsByEntities(db *sql.DB, entityIDs []string) []string {
	var ids []string
	for _, eid := range entityIDs {
		rows, err := db.Query(`SELECT prisoner_id FROM prisoner_entity WHERE entity_id = ?;`, eid)
		if err != nil {
			log.Println("查询 prisoner_id 失败:", err)
			continue
		}
		defer rows.Close()

		for rows.Next() {
			var pid string
			rows.Scan(&pid)
			ids = append(ids, pid)
		}
	}
	return unique(ids)
}

// getUserIDsByPrisoners 根据 prisoner_id 查询对应的 user_id
func getUserIDsByPrisoners(db *sql.DB, prisonerIDs []string) []string {
	var ids []string
	for _, pid := range prisonerIDs {
		rows, err := db.Query(`SELECT user_id FROM user_profile WHERE prisoner_id = ?;`, pid)
		if err != nil {
			log.Println("查询 user_id 失败:", err)
			continue
		}
		defer rows.Close()

		for rows.Next() {
			var uid string
			rows.Scan(&uid)
			ids = append(ids, uid)
		}
	}
	return unique(ids)
}

// unique 去重函数
func unique(list []string) []string {
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
