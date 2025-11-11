package DBwatcher

import (
	"database/sql"
	"fmt"
)

// GetGenderAndSize 根据 steamID 查询性别和尺寸
// db: 已打开的 *sql.DB 对象
// 返回 gender ("F" 或 "M") 和尺寸（float64），如果查询失败返回 "", 0 和错误
func GetGenderAndSize(db *sql.DB, steamID string) (string, float64, error) {
	var userProfileID int64

	// 1️⃣ 在 user_profile 表查询 steamID 对应的 id
	err := db.QueryRow(`SELECT id FROM user_profile WHERE user_id = ?`, steamID).Scan(&userProfileID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", 0, fmt.Errorf("steamID %s 未找到对应 user_profile", steamID)
		}
		return "", 0, fmt.Errorf("查询 user_profile 出错: %v", err)
	}

	// 2️⃣ 在 prisoner 表查询 gender
	var gender int
	err = db.QueryRow(`SELECT gender FROM prisoner WHERE user_profile_id = ?`, userProfileID).Scan(&gender)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", 0, fmt.Errorf("user_profile_id %d 未找到对应 prisoner", userProfileID)
		}
		return "", 0, fmt.Errorf("查询 prisoner gender 出错: %v", err)
	}

	// 3️⃣ 根据 gender 查询对应尺寸
	if gender == 1 {
		var breastSize float64
		err = db.QueryRow(`SELECT breast_size FROM prisoner WHERE user_profile_id = ?`, userProfileID).Scan(&breastSize)
		if err != nil {
			return "", 0, fmt.Errorf("查询 breast_size 出错: %v", err)
		}
		return "F", breastSize, nil
	} else if gender == 2 {
		var penisSize float64
		err = db.QueryRow(`SELECT penis_size FROM prisoner WHERE user_profile_id = ?`, userProfileID).Scan(&penisSize)
		if err != nil {
			return "", 0, fmt.Errorf("查询 penis_size 出错: %v", err)
		}
		return "M", penisSize, nil
	} else {
		return "", 0, fmt.Errorf("未知 gender 值: %d", gender)
	}
}
