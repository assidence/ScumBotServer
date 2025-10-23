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
func ParseTradeLog(line string) *TradeEvent {
	line = strings.TrimSpace(line)

	// 通用前缀匹配：提取时间戳与模块名
	prefix := regexp.MustCompile(`^(\d{4}\.\d{2}\.\d{2}-\d{2}\.\d{2}\.\d{2}): \[(Trade|Trade-Mechanic|Currency Conversion)\] (.*)`)
	m := prefix.FindStringSubmatch(line)
	if len(m) != 4 {
		return nil
	}
	timestampStr, category, content := m[1], m[2], m[3]
	timestamp, _ := time.Parse("2006.01.02-15.04.05", timestampStr)

	switch category {
	case "Trade":
		return parseClassicTrade(timestamp, content)
	case "Trade-Mechanic":
		return parseMechanicTrade(timestamp, content)
	case "Currency Conversion":
		return parseCurrencyConversion(timestamp, content)
	default:
		return nil
	}
}

// ==================== 子解析器 ====================

// 普通交易（购买/出售）
func parseClassicTrade(ts time.Time, content string) *TradeEvent {
	re := regexp.MustCompile(`Tradeable \(([^)]+)\) (purchased|sold) by ([^(]+)\((\d+)\) for ([\d\.]+).*trader (\S+)`)
	m := re.FindStringSubmatch(content)
	if len(m) != 7 {
		return nil
	}
	price, _ := strconv.ParseFloat(m[5], 64)
	return &TradeEvent{
		Timestamp: ts,
		Type:      "Trade",
		Action:    m[2],
		Player:    strings.TrimSpace(m[3]),
		PlayerID:  m[4],
		ItemName:  m[1],
		Price:     price,
		Trader:    m[6],
	}
}

// 载具交易（修理、加油、喷漆、卖出）
func parseMechanicTrade(ts time.Time, content string) *TradeEvent {
	re := regexp.MustCompile(`Vehicle.*(repaired|refueled|painted|sold|purchased) by ([^(]+)\((\d+)\) for ([\d\.]+).*trader (\S+)`)
	m := re.FindStringSubmatch(content)
	if len(m) != 6 {
		return nil
	}
	price, _ := strconv.ParseFloat(m[4], 64)
	return &TradeEvent{
		Timestamp: ts,
		Type:      "Trade-Mechanic",
		Action:    m[1],
		Player:    strings.TrimSpace(m[2]),
		PlayerID:  m[3],
		Price:     price,
		Trader:    m[5],
	}
}

// 货币兑换（银行兑换）
func parseCurrencyConversion(ts time.Time, content string) *TradeEvent {
	re := regexp.MustCompile(`Player ([^(]+)\((\d+)\) converted ([\d\.]+) (\w+) to ([\d\.]+) (\w+)`)
	m := re.FindStringSubmatch(content)
	if len(m) != 7 {
		return nil
	}
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
