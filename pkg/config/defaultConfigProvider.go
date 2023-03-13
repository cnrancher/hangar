package config

import (
	"sync"

	"github.com/sirupsen/logrus"
)

type defaultConfigProvider struct {
	mu   sync.RWMutex
	data map[string]any
}

var DefaultProvider Provider = &defaultConfigProvider{
	data: make(map[string]any),
}

func (c *defaultConfigProvider) Get(key string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if key == "" {
		dst := make(map[string]any)
		for k, v := range c.data {
			dst[k] = v
		}
		return dst
	}
	return c.data[key]
}

func (c *defaultConfigProvider) GetString(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v := c.data[key]
	switch v := v.(type) {
	case string:
		return v
	}
	return ""
}

func (c *defaultConfigProvider) GetStringSlice(key string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v := c.data[key]
	switch v := v.(type) {
	case []string:
		if v == nil {
			return nil
		}
		vv := make([]string, len(v))
		copy(vv, v)
		return vv
	}
	return nil
}

func (c *defaultConfigProvider) GetInt(key string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v := c.data[key]
	switch v := v.(type) {
	case int:
		return v
	}
	return 0
}

func (c *defaultConfigProvider) GetBool(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v := c.data[key]
	switch v := v.(type) {
	case bool:
		return v
	}
	return false
}

func (c *defaultConfigProvider) Set(key string, value any) {
	if key == "" {
		return
	}
	switch value.(type) {
	case int:
	case string:
	case []string:
	case bool:
	default:
		logrus.Warnf("unable to set %v to config: invalid value type %T",
			value, value)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data == nil {
		c.data = make(map[string]any)
	}
	c.data[key] = value
}

func (c *defaultConfigProvider) IsSet(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.data[key]
	return ok
}

func Get(k string) any {
	return DefaultProvider.Get(k)
}

func GetInt(k string) int {
	return DefaultProvider.GetInt(k)
}

func GetBool(k string) bool {
	return DefaultProvider.GetBool(k)
}

func GetString(k string) string {
	return DefaultProvider.GetString(k)
}

func GetStringSlice(k string) []string {
	return DefaultProvider.GetStringSlice(k)
}

func Set(k string, v any) {
	DefaultProvider.Set(k, v)
}

func IsSet(k string) bool {
	return DefaultProvider.IsSet(k)
}
