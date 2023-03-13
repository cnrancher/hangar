package config_test

import (
	"reflect"
	"testing"

	"github.com/cnrancher/hangar/pkg/config"
)

func init() {
	config.Set("string", "value")
	config.Set("int", 1)
	config.Set("bool", true)
	config.Set("stringSlice", []string{"a", "b"})
}

func Test_Get(t *testing.T) {
	// when key is "", get the config data map
	v := config.Get("")
	switch v := v.(type) {
	case map[string]any:
		// check data
		if v["int"].(int) != 1 {
			t.Error("failed")
			return
		}
		v["int"] = 2
		if config.GetInt("int") != 1 {
			t.Error("failed")
		}
	default:
		t.Error("failed")
		return
	}

	v = config.Get("int")
	switch v := v.(type) {
	case int:
		if v != 1 {
			t.Error("failed")
		}
	default:
		t.Error("failed")
	}
}

func Test_GetString(t *testing.T) {
	v := config.GetString("string")
	if v != "value" {
		t.Error("failed")
	}
	v = config.GetString("aaa")
	if v != "" {
		t.Error("failed")
	}
}

func Test_GetStringSlice(t *testing.T) {
	v := config.GetStringSlice("stringSlice")
	if !reflect.DeepEqual(v, []string{"a", "b"}) {
		t.Error("failed")
	}
	v = config.GetStringSlice("aaa")
	if v != nil {
		t.Errorf("failed: %++v\n", v)
	}
}

func Test_Int(t *testing.T) {
	v := config.GetInt("int")
	if v != 1 {
		t.Error("failed")
	}
	if config.GetInt("aaa") != 0 {
		t.Error("failed")
	}
}

func Test_GetBool(t *testing.T) {
	v := config.GetBool("bool")
	if !v {
		t.Error("failed")
	}
	if config.GetBool("aaa") {
		t.Error("failed")
	}
}

func Test_IsSet(t *testing.T) {
	config.Set("key", "value")
	if !config.IsSet("key") {
		t.Error("failed")
	}
	if config.IsSet("aaa") {
		t.Error("failed")
	}
}
