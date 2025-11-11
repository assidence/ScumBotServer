package PlayersInfo

import (
	"fmt"
	"gopkg.in/ini.v1"
	"log"
	"strconv"
	"strings"
)

// Debug 输出管理
var PCDebugEnabled = false // 可以在模块初始化或运行时控制开关

func pcdebug(format string, a ...interface{}) {
	if PCDebugEnabled {
		log.Printf("[PlayersInfo-DEBUG] "+format, a...)
	}
}

// -------------------- 条件结构 --------------------
type Condition struct {
	Operator string
	Value    float64
}

type PlayerConditionGroup struct {
	Attributes map[string]Condition
	Skills     map[string]Condition
}

// -------------------- INI 条件加载 --------------------

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

		pcdebug("== 加载条件组: %s ==", section.Name())

		for _, key := range section.Keys() {
			rawKey := strings.TrimSpace(key.Name())
			rawVal := strings.TrimSpace(key.Value())

			attrName, op, val, perr := parseConditionFromKeyOrValue(rawKey, rawVal)
			if perr != nil {
				pcdebug("[PlayerInfo-Error] 无法解析条件: %s=%s (%v)", rawKey, rawVal, perr)
				continue
			}

			pcdebug("  条件: %s %s %v", attrName, op, val)

			if strings.Contains(attrName, "Skill") {
				group.Skills[attrName] = Condition{Operator: op, Value: val}
			} else {
				group.Attributes[attrName] = Condition{Operator: op, Value: val}
			}
		}

		allGroups[section.Name()] = group
	}

	pcGroups = allGroups
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
		op = "=="
	}

	v, err := strconv.ParseFloat(strings.TrimSpace(valStr), 64)
	if err != nil {
		return "", "", 0, fmt.Errorf("value parse error: %w", err)
	}
	return strings.TrimSpace(rawKey), op, v, nil
}

// compare: 只支持 > < ==
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

// -------------------- 玩家条件判断 --------------------

// PlayerAchievementTrick 判断玩家是否符合条件组
func PlayerAchievementTrick(Players map[string]interface{}) map[string][]string {
	result := make(map[string][]string)

	for steamID, playerIface := range Players {
		playerMap, ok := playerIface.(map[string]interface{})
		if !ok {
			pcdebug("[PlayersInfo-Error] 类型断言失败: %s", steamID)
			continue
		}

		for groupName, group := range pcGroups {
			match := true

			if attrs, ok := playerMap["Attributes"].(map[string]interface{}); ok {
				for attrName, cond := range group.Attributes {
					valIface, exists := attrs[attrName]
					if !exists {
						pcdebug("[属性不匹配] %s %s %v (不存在)", attrName, cond.Operator, cond.Value)
						match = false
						break
					}
					val, ok := valIface.(float64)
					if !ok || !compare(val, cond.Operator, cond.Value) {
						pcdebug("[属性不匹配] %s %s %v (玩家值=%v)", attrName, cond.Operator, cond.Value, valIface)
						match = false
						break
					} else {
						pcdebug("[属性匹配] %s %s %v (玩家值=%v)", attrName, cond.Operator, cond.Value, val)
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
									pcdebug("[技能不匹配] %s %s %v (玩家Level=%v)", skillName, cond.Operator, cond.Value, level)
									match = false
								} else {
									pcdebug("[技能匹配] %s %s %v (玩家Level=%v)", skillName, cond.Operator, cond.Value, level)
								}
								found = true
								break
							}
						}
					}
					if !found {
						pcdebug("[技能不匹配] %s 不存在", skillName)
						match = false
					}
				}
			}

			if match {
				pcdebug("[匹配成功] 玩家符合条件组: %s", groupName)
				result[groupName] = append(result[groupName], steamID)
			} else {
				pcdebug("[匹配失败] 玩家不符合条件组: %s", groupName)
			}
		}
	}

	return result
}
