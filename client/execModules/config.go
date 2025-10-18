package execModules

import (
	"fmt"
	"gopkg.in/ini.v1"
	"strconv"
	"strings"
	"sync"
)

type Config struct {
	filePath string
	Data     map[string]map[string]interface{}
	lock     sync.RWMutex
}

// NewConfig 创建对象并加载文件
func NewConfig(path string) (*Config, error) {
	cfg := &Config{
		filePath: path,
		Data:     make(map[string]map[string]interface{}),
	}
	if err := cfg.Load(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Load 读取 INI 并自动解析类型
func (c *Config) Load() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	iniCfg, err := ini.Load(c.filePath)
	if err != nil {
		return fmt.Errorf("读取配置失败: %v", err)
	}

	tempData := make(map[string]map[string]interface{})

	for _, section := range iniCfg.Sections() {
		if section.Name() == "DEFAULT" { // 跳过 DEFAULT
			continue
		}
		secMap := make(map[string]interface{})
		for _, key := range section.Keys() {
			secMap[key.Name()] = parseValue(key.String())
		}
		tempData[section.Name()] = secMap
	}

	c.Data = tempData
	return nil
}

// parseValue 自动识别 int / float / bool / string
func parseValue(val string) interface{} {
	val = strings.TrimSpace(val)

	// 先尝试 int
	if i, err := strconv.ParseInt(val, 10, 64); err == nil {
		return i
	}

	// 再尝试 float
	if f, err := strconv.ParseFloat(val, 64); err == nil {
		return f
	}

	// 最后尝试 bool（仅解析 true/false，不解析 1/0）
	lowerVal := strings.ToLower(val)
	if lowerVal == "true" {
		return true
	}
	if lowerVal == "false" {
		return false
	}

	// 默认 string
	return val
}

// GetValue 获取任意值（interface{}）
func (c *Config) GetValue(section, key string) interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if sec, ok := c.Data[section]; ok {
		if val, ok := sec[key]; ok {
			return val
		}
	}
	return nil
}

// GetString
func (c *Config) GetString(section, key string) string {
	v := c.GetValue(section, key)
	if str, ok := v.(string); ok {
		return str
	}
	return fmt.Sprint(v)
}

// GetInt
func (c *Config) GetInt(section, key string) int64 {
	v := c.GetValue(section, key)
	if i, ok := v.(int64); ok {
		return i
	}
	if f, ok := v.(float64); ok {
		return int64(f)
	}
	return 0
}

// GetFloat
func (c *Config) GetFloat(section, key string) float64 {
	v := c.GetValue(section, key)
	if f, ok := v.(float64); ok {
		return f
	}
	if i, ok := v.(int64); ok {
		return float64(i)
	}
	return 0
}

// GetBool
func (c *Config) GetBool(section, key string) bool {
	v := c.GetValue(section, key)
	if b, ok := v.(bool); ok {
		return b
	}
	// 其他类型尝试字符串解析
	return strings.ToLower(fmt.Sprint(v)) == "true"
}

// SetValue 修改或新增值
func (c *Config) SetValue(section, key string, value interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if _, ok := c.Data[section]; !ok {
		c.Data[section] = make(map[string]interface{})
	}
	c.Data[section][key] = value
}

// Save 保存回 INI 文件
func (c *Config) Save() error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	iniCfg := ini.Empty()
	for section, keys := range c.Data {
		sec := iniCfg.Section(section)
		for k, v := range keys {
			sec.Key(k).SetValue(fmt.Sprint(v))
		}
	}

	return iniCfg.SaveTo(c.filePath)
}
