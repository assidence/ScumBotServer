package modules

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type TradeEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`   // Trade / Trade-Mechanic / Currency Conversion
	Action    string    `json:"action"` // sold / purchased / convert / repaired ...
	Player    string    `json:"player"`
	PlayerID  string    `json:"player_id"`
	ItemName  string    `json:"item_name"`
	Amount    int       `json:"amount"`
	Price     float64   `json:"price"`
	Trader    string    `json:"trader"`
	Extra     string    `json:"extra"`
}

// ParseTradeLog 从日志文本中解析交易行为
func EconomyHandler(ecoch chan string, execch chan string) {
	// 通用前缀匹配：提取时间戳与模块名
	prefix := regexp.MustCompile(`^(\d{4}\.\d{2}\.\d{2}-\d{2}\.\d{2}\.\d{2}): \[(Trade|Trade-Mechanic|Currency Conversion|Bank)\] (.*)`)
	execData := map[string]string{
		"steamID":     "000000",
		"nickName":    "System",
		"command":     "0",
		"commandArgs": "0",
	}
	var jsonByte []byte
	for line := range ecoch {
		line = strings.TrimSpace(line)

		m := prefix.FindStringSubmatch(line)
		if len(m) != 4 {
			fmt.Println("[Economy-Error] Insufficient elements matched.")
			continue
		}
		timestampStr, category, content := m[1], m[2], m[3]
		timestamp, _ := time.Parse("2006.01.02-15.04.05", timestampStr)

		switch category {
		case "Trade":
			result := parseClassicTrade(timestamp, content)
			//fmt.Println(result)
			if result == nil {
				fmt.Println("[Economy-Error] No Trade event：", content)
				continue
			}
			execData["command"] = result.Action
			execData["commandArgs"] = fmt.Sprintf("%s-%s-%d", result.PlayerID, result.ItemName, result.Amount)
		case "Trade-Mechanic":
			result := parseMechanicTrade(timestamp, content)
			//fmt.Println(result)
			if result == nil {
				fmt.Println("[Economy-Error] No Trade-Mechanic event")
				continue
			}
			execData["command"] = result.Action
			execData["commandArgs"] = fmt.Sprintf("%s-%s-%d", result.PlayerID, result.ItemName, result.Amount)
		case "Currency Conversion":
			result := parseCurrencyConversion(timestamp, content)
			//fmt.Println(result)
			if result == nil {
				fmt.Println("[Economy-Error] No Currency Conversion event")
				continue
			}
			execData["command"] = result.Action
			execData["commandArgs"] = fmt.Sprintf("%s-%s-%d", result.PlayerID, result.ItemName, result.Amount)
		case "Bank":
			result := parseBankEvent(timestamp, content)
			if result == nil {
				fmt.Println("[Economy-Error] No Bank event")
				continue
			}
			execData["command"] = result.Action
			execData["commandArgs"] = fmt.Sprintf("%s-%s-%d", result.PlayerID, result.ItemName, result.Amount)
		default:
			//fmt.Println("[Economy-Module] Unknown category.")
			continue
		}
		jsonByte = sequenceJson(&execData)
		execch <- string(jsonByte)

	}
}

// ==================== 子解析器 ====================

// 处理普通交易（购买/出售）
func parseClassicTrade(ts time.Time, content string) *TradeEvent {
	re := regexp.MustCompile(`Tradeable \((.+?)\) (purchased|sold) by ([^(]+)\((\d+)\) for ([\d\.]+)(?: \([^)]*worth of contained items\))?(?: money)? (?:from|to) trader (\S+)`)
	m := re.FindStringSubmatch(content)
	if len(m) != 7 {
		fmt.Println("[Economy-Error] parseClassicTrade: insufficient match:", content)
		return nil
	}

	price, _ := strconv.ParseFloat(m[5], 64)

	return &TradeEvent{
		Timestamp: ts,
		Type:      "Trade",
		Action:    m[2],
		Player:    strings.TrimSpace(m[3]),
		PlayerID:  m[4],
		ItemName:  strings.ReplaceAll(m[1], "-", "_"),
		Price:     price,
		Trader:    m[6],
	}
}

// 载具交易（修理、加油、喷漆、卖出）
func parseMechanicTrade(ts time.Time, content string) *TradeEvent {
	re := regexp.MustCompile(`Service \((.+?)\) (repaired|refueled|painted|sold|purchased) by (.+?)\((\d+)\) for ([\d\.]+) money from trader (\S+)`)
	m := re.FindStringSubmatch(content)
	if len(m) != 7 {
		fmt.Println("[Economy-Error] parseMechanicTrade: insufficient match:", content)
		return nil
	}
	price, _ := strconv.ParseFloat(m[5], 64)
	return &TradeEvent{
		Timestamp: ts,
		Type:      "Trade-Mechanic",
		Action:    m[2],
		Player:    strings.TrimSpace(m[3]),
		PlayerID:  m[4],
		ItemName:  strings.ReplaceAll(m[1], "-", "_"),
		Price:     price,
		Trader:    m[6],
	}
}

// 货币兑换（银行兑换）
func parseCurrencyConversion(ts time.Time, content string) *TradeEvent {
	// 先使用原来的正则
	oldRe := regexp.MustCompile(`Player ([^(]+)\((\d+)\) converted ([\d\.]+) (\w+) to ([\d\.]+) (\w+)`)
	m := oldRe.FindStringSubmatch(content)
	if len(m) == 7 {
		fromAmt, _ := strconv.ParseFloat(m[3], 64)
		toAmt, _ := strconv.ParseFloat(m[5], 64)
		return &TradeEvent{
			Timestamp: ts,
			Type:      "Currency Conversion",
			Action:    "convert",
			Player:    strings.TrimSpace(m[1]),
			PlayerID:  m[2],
			ItemName:  fmt.Sprintf("%s→%s", m[4], m[6]),
			Price:     fromAmt,
			Extra:     fmt.Sprintf("%.2f→%.2f", fromAmt, toAmt),
		}
	}

	// 如果老正则不匹配，再尝试新的purchased...for...格式
	newRe := regexp.MustCompile(`(.+?)\(ID:(\d+)\)\(Account Number:(\d+)\) purchased (\d+) (\w+) for ([\d\.]+) (\w+)`)
	m2 := newRe.FindStringSubmatch(content)
	if len(m2) == 8 {
		amount, _ := strconv.Atoi(m2[4])
		price, _ := strconv.ParseFloat(m2[6], 64)
		return &TradeEvent{
			Timestamp: ts,
			Type:      "Currency Conversion",
			Action:    "purchase",
			Player:    strings.TrimSpace(m2[1]),
			PlayerID:  m2[2],
			ItemName:  m2[5],
			Amount:    amount,
			Price:     price,
			Extra:     fmt.Sprintf("Paid %s %s", m2[6], m2[7]),
		}
	}

	// 都不匹配返回nil
	fmt.Println("[Economy-Error] parseCurrencyConversion: no match", content)
	return nil
}

func parseBankEvent(ts time.Time, content string) *TradeEvent {
	// 解析 deposit
	depositRe := regexp.MustCompile(`(.+?)\(ID:(\d+)\)\(Account Number:(\d+)\) deposited (\d+)(?:\((\d+) was added\))? to Account Number: (\d+).+`)
	m := depositRe.FindStringSubmatch(content)
	if len(m) == 7 {
		amount, _ := strconv.Atoi(m[4])
		return &TradeEvent{
			Timestamp: ts,
			Type:      "Bank",
			Action:    "deposit",
			Player:    strings.TrimSpace(m[1]),
			PlayerID:  m[2],
			ItemName:  "Money",
			Amount:    amount,
			Extra:     fmt.Sprintf("added %s to account %s", m[5], m[6]),
		}
	}

	// 解析 destroyed card
	cardRe := regexp.MustCompile(`(.+?)\(ID:(\d+)\)\(Account Number:(\d+)\) manually destroyed (.+) belonging to Account Number:(\d+).+`)
	m2 := cardRe.FindStringSubmatch(content)
	if len(m2) == 6 {
		return &TradeEvent{
			Timestamp: ts,
			Type:      "Bank",
			Action:    "destroyed_card",
			Player:    strings.TrimSpace(m2[1]),
			PlayerID:  m2[2],
			ItemName:  strings.ReplaceAll(m2[4], "-", "_"),
			Extra:     fmt.Sprintf("Account Number:%s", m2[5]),
		}
	}

	fmt.Println("[Economy-Error] parseBankEvent: no match for content:", content)
	return nil
}
