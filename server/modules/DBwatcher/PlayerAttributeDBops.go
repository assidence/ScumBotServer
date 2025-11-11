package DBwatcher

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

const (
	BODY_SIM_KEY_PADDING   = 5
	BODY_SIM_VALUE_PADDING = 10
)

// ========== 数据结构 ==========
type PlayerRow struct {
	PrisonerID int
	Name       string
	SteamID    string
}

type Skill struct {
	Name       string
	Level      float64
	Experience float64
}

type PlayerFullInfo struct {
	PlayerRow
	Attributes    map[string]float64
	Skills        []Skill
	GenderAndSize map[string]float64
}

var attributeNames = []string{
	"BaseStrength",
	"BaseConstitution",
	"BaseDexterity",
	"BaseIntelligence",
}

// ========== 内部工具函数 ==========

// 获取 body_simulation blob
func getBodySim(db *sql.DB, prisonerID int) ([]byte, error) {
	row := db.QueryRow("SELECT body_simulation FROM prisoner WHERE id = ?", prisonerID)
	var blob []byte
	if err := row.Scan(&blob); err != nil {
		return nil, err
	}
	return blob, nil
}

// 查找单个属性
func findAttribute(bodySim []byte, key string) (float64, error) {
	keyBytes := []byte(key)
	idx := bytes.Index(bodySim, keyBytes)
	if idx == -1 {
		return 0, fmt.Errorf("attribute %s not found", key)
	}

	propOffset := idx + len(keyBytes) + BODY_SIM_KEY_PADDING
	propName := []byte("DoubleProperty")
	if propOffset+len(propName) > len(bodySim) {
		return 0, fmt.Errorf("property name out of range for %s", key)
	}
	if !bytes.Equal(bodySim[propOffset:propOffset+len(propName)], propName) {
		return 0, fmt.Errorf("unexpected property type for %s", key)
	}

	valOffset := propOffset + len(propName) + BODY_SIM_VALUE_PADDING
	if valOffset+8 > len(bodySim) {
		return 0, fmt.Errorf("value offset out of range for %s", key)
	}

	valBytes := bodySim[valOffset : valOffset+8]
	var val float64
	if err := binary.Read(bytes.NewReader(valBytes), binary.LittleEndian, &val); err != nil {
		return 0, err
	}
	return val, nil
}

// 从 SteamID 查找 prisoner 信息
func getPlayerBySteamID(db *sql.DB, steamID string) (*PlayerRow, error) {
	row := db.QueryRow(`
		SELECT prisoner.id AS prisoner_id,
		       COALESCE(user_profile.name, '') AS name,
		       user_profile.user_id
		FROM prisoner
		JOIN user_profile ON prisoner.user_profile_id = user_profile.id
		WHERE user_profile.user_id = ?
	`, steamID)

	var r PlayerRow
	var userID sql.NullString
	if err := row.Scan(&r.PrisonerID, &r.Name, &userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("未找到 steamid=%s 对应的玩家", steamID)
		}
		return nil, err
	}
	if userID.Valid {
		r.SteamID = userID.String
	}
	return &r, nil
}

// 读取属性
func readPlayerAttributes(db *sql.DB, prisonerID int) (map[string]float64, error) {
	bodySim, err := getBodySim(db, prisonerID)
	if err != nil {
		return nil, err
	}
	res := make(map[string]float64)
	for _, a := range attributeNames {
		v, err := findAttribute(bodySim, a)
		if err != nil {
			res[a] = -1
		} else {
			res[a] = v
		}
	}
	return res, nil
}

// 读取技能
func readPlayerSkills(db *sql.DB, prisonerID int) ([]Skill, error) {
	rows, err := db.Query("SELECT name, level, experience FROM prisoner_skill WHERE prisoner_id = ?", prisonerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []Skill
	for rows.Next() {
		var name string
		var level sql.NullFloat64
		var exp sql.NullFloat64
		if err := rows.Scan(&name, &level, &exp); err != nil {
			return nil, err
		}

		s := Skill{Name: name}
		if level.Valid {
			s.Level = level.Float64
		}
		if exp.Valid {
			s.Experience = exp.Float64
		}
		skills = append(skills, s)
	}
	return skills, nil
}

// ========== 对外公开函数 ==========

// 自动重试的封装，防止 database is locked
func GetPlayerFullInfo(db *sql.DB, steamID string) (*PlayerFullInfo, error) {
	var lastErr error
	for i := 0; i < 3; i++ {
		info, err := getPlayerFullInfoOnce(db, steamID)
		if err == nil {
			return info, nil
		}
		lastErr = err
		if !strings.Contains(err.Error(), "database is locked") {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil, lastErr
}

// 内部真正的读取逻辑（使用外部传入的 db）
func getPlayerFullInfoOnce(db *sql.DB, steamID string) (*PlayerFullInfo, error) {
	player, err := getPlayerBySteamID(db, steamID)
	if err != nil {
		return nil, err
	}

	attrs, err := readPlayerAttributes(db, player.PrisonerID)
	if err != nil {
		return nil, fmt.Errorf("读取属性失败: %v", err)
	}

	skills, err := readPlayerSkills(db, player.PrisonerID)
	if err != nil {
		return nil, fmt.Errorf("读取技能失败: %v", err)
	}

	return &PlayerFullInfo{
		PlayerRow:  *player,
		Attributes: attrs,
		Skills:     skills,
	}, nil
}

//var info *PlayerFullInfo

// 获取“头尖尖”玩家列表：BaseStrength≥7.9 且 BaseDexterity≥4.9
func GetStrongPlayers(db *sql.DB, userIDs []string) []string {
	var strongPlayers []string

	for _, steamID := range userIDs {
		//var err error
		info, err := getPlayerFullInfoOnce(db, steamID)
		if err != nil {
			// 忽略未找到或读取失败的玩家（比如离线、数据库未同步等）
			continue
		}

		strength := info.Attributes["BaseStrength"]
		dexterity := info.Attributes["BaseDexterity"]

		if strength >= 7.9 && dexterity >= 4.9 {
			strongPlayers = append(strongPlayers, steamID)
		}
	}

	return strongPlayers
}

// 根据多个 SteamID 查询玩家信息
func GetPlayerFullInfoByList(db *sql.DB, userIDs []string) map[string]*PlayerFullInfo {
	result := make(map[string]*PlayerFullInfo)

	for _, steamID := range userIDs {
		info, err := getPlayerFullInfoOnce(db, steamID)
		if err != nil {
			// 忽略未找到或读取失败的玩家
			continue
		}
		result[steamID] = info
	}

	return result
}
