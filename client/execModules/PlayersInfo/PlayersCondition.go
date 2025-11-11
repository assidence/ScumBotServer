package PlayersInfo

import (
	"fmt"
	"gopkg.in/ini.v1"
	"log"
	"strconv"
	"strings"
)

type Condition struct {
	Operator string
	Value    float64
}

type PlayerConditionGroup struct {
	Attributes map[string]Condition
	Skills     map[string]Condition
}

// LoadPlayerCondition: 支持 key 或 value 两种写法，只允许 > < ==
func LoadPlayerCondition(path string) map[string]*PlayerConditionGroup {
	cfg, err := ini.Load(path)
	if err != nil {
		log.Fatalf("[PlayerInfo-Error] 无法加载 ini 文件: %v", err)
	}

	allGroups := make(map[string]*PlayerConditionGroup)

	for _, section := range cfg.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}

		group := &PlayerConditionGroup{
			Attributes: make(map[string]Condition),
			Skills:     make(map[string]Condition),
		}

		fmt.Printf("== 加载条件组: %s ==\n", section.Name())

		for _, key := range section.Keys() {
			rawKey := strings.TrimSpace(key.Name())
			rawVal := strings.TrimSpace(key.Value())

			attrName, op, val, perr := parseConditionFromKeyOrValue(rawKey, rawVal)
			if perr != nil {
				fmt.Printf("[PlayerInfo-Error] 无法解析条件: %s=%s (%v)\n", rawKey, rawVal, perr)
				continue
			}

			fmt.Printf("  条件: %s %s %v\n", attrName, op, val)

			if strings.Contains(attrName, "Skill") {
				group.Skills[attrName] = Condition{Operator: op, Value: val}
			} else {
				group.Attributes[attrName] = Condition{Operator: op, Value: val}
			}
		}

		allGroups[section.Name()] = group
	}

	//pcGroups = allGroups
	return allGroups
}

// parseConditionFromKeyOrValue: 只支持 > < ==
func parseConditionFromKeyOrValue(rawKey, rawVal string) (string, string, float64, error) {
	tryTrimKey := func(k string) (string, string, bool) {
		if strings.HasSuffix(k, ">") {
			return strings.TrimSuffix(k, ">"), ">", true
		}
		if strings.HasSuffix(k, "<") {
			return strings.TrimSuffix(k, "<"), "<", true
		}
		if strings.HasSuffix(k, "==") {
			return strings.TrimSuffix(k, "=="), "==", true
		}
		return k, "", false
	}

	cleanKey, op, found := tryTrimKey(rawKey)
	if found {
		v, err := strconv.ParseFloat(strings.TrimSpace(rawVal), 64)
		if err != nil {
			return "", "", 0, fmt.Errorf("value parse error: %w", err)
		}
		return strings.TrimSpace(cleanKey), op, v, nil
	}

	valStr := strings.TrimSpace(rawVal)
	switch {
	case strings.HasPrefix(valStr, ">"):
		op = ">"
		valStr = valStr[1:]
	case strings.HasPrefix(valStr, "<"):
		op = "<"
		valStr = valStr[1:]
	case strings.HasPrefix(valStr, "=="):
		op = "=="
		valStr = valStr[2:]
	default:
		// 没有显式操作符，默认 ==
		op = "=="
	}

	v, err := strconv.ParseFloat(strings.TrimSpace(valStr), 64)
	if err != nil {
		return "", "", 0, fmt.Errorf("value parse error: %w", err)
	}
	return strings.TrimSpace(rawKey), op, v, nil
}

// 对比函数：只支持 > < ==
func compare(val float64, op string, target float64) bool {
	switch op {
	case ">":
		return val > target
	case "<":
		return val < target
	case "==":
		return val == target
	default:
		return false
	}
}

// 判断玩家是否符合条件组
func PlayerAchievementTrick(Players map[string]interface{}) map[string][]string {
	result := make(map[string][]string)

	for steamID, playerIface := range Players {
		playerMap, ok := playerIface.(map[string]interface{})
		if !ok {
			//fmt.Println("[PlayersInfo-Error] 类型断言失败:", steamID)
			continue
		}
		/*
			fmt.Printf("====== 玩家SteamID: %s ======\n", steamID)

			if attrs, ok := playerMap["Attributes"].(map[string]interface{}); ok {
				fmt.Println("属性:")
				for k, v := range attrs {
					fmt.Printf("  %s = %v\n", k, v)
				}
			}

			if skills, ok := playerMap["Skills"].([]interface{}); ok {
				fmt.Println("技能:")
				for _, s := range skills {
					if skillMap, ok := s.(map[string]interface{}); ok {
						fmt.Printf("  %s - Level: %v, Exp: %v\n",
							skillMap["Name"], skillMap["Level"], skillMap["Experience"])
					}
				}
			}


		*/
		for groupName, group := range pcGroups {
			//fmt.Printf("-- 检查条件组: %s --\n", groupName)
			match := true

			if attrs, ok := playerMap["Attributes"].(map[string]interface{}); ok {
				for attrName, cond := range group.Attributes {
					valIface, exists := attrs[attrName]
					if !exists {
						//fmt.Printf("[属性不匹配] %s %s %v (不存在)\n", attrName, cond.Operator, cond.Value)
						match = false
						break
					}
					val, ok := valIface.(float64)
					if !ok || !compare(val, cond.Operator, cond.Value) {
						//fmt.Printf("[属性不匹配] %s %s %v (玩家值=%v)\n", attrName, cond.Operator, cond.Value, val)
						match = false
						break
					} else {
						//fmt.Printf("[属性匹配] %s %s %v (玩家值=%v)\n", attrName, cond.Operator, cond.Value, val)
					}
				}
			} else {
				match = false
			}

			if skills, ok := playerMap["Skills"].([]interface{}); ok {
				for skillName, cond := range group.Skills {
					found := false
					for _, s := range skills {
						if skillMap, ok := s.(map[string]interface{}); ok {
							if skillMap["Name"].(string) == skillName {
								level := skillMap["Level"].(float64)
								if !compare(level, cond.Operator, cond.Value) {
									//fmt.Printf("[技能不匹配] %s %s %v (玩家Level=%v)\n", skillName, cond.Operator, cond.Value, level)
									match = false
								} else {
									//fmt.Printf("[技能匹配] %s %s %v (玩家Level=%v)\n", skillName, cond.Operator, cond.Value, level)
								}
								found = true
								break
							}
						}
					}
					if !found {
						//fmt.Printf("[技能不匹配] %s 不存在\n", skillName)
						match = false
					}
				}
			}

			if match {
				//fmt.Printf("[匹配成功] 玩家符合条件组: %s\n", groupName)
				result[groupName] = append(result[groupName], steamID)
			} else {
				//fmt.Printf("[匹配失败] 玩家不符合条件组: %s\n", groupName)
			}
		}

		//fmt.Println("====== 玩家检查完毕 ======")
	}

	return result
}
