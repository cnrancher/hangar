package cmdconfig_test

import (
	"reflect"
	"testing"

	"github.com/cnrancher/hangar/pkg/cmdconfig"
)

func init() {
	cmdconfig.Set("string", "value")
	cmdconfig.Set("int", 1)
	cmdconfig.Set("bool", true)
	cmdconfig.Set("stringSlice", []string{"a", "b"})
}

func Test_Get(t *testing.T) {
	// when key is "", get the cmdconfig data map
	v := cmdconfig.Get("")
	switch v := v.(type) {
	case map[string]any:
		// check data
		if v["int"].(int) != 1 {
			t.Error("failed")
			return
		}
		v["int"] = 2
		if cmdconfig.GetInt("int") != 1 {
			t.Error("failed")
		}
	default:
		t.Error("failed")
		return
	}

	v = cmdconfig.Get("int")
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
	v := cmdconfig.GetString("string")
	if v != "value" {
		t.Error("failed")
	}
	v = cmdconfig.GetString("aaa")
	if v != "" {
		t.Error("failed")
	}
}

func Test_GetStringSlice(t *testing.T) {
	v := cmdconfig.GetStringSlice("stringSlice")
	if !reflect.DeepEqual(v, []string{"a", "b"}) {
		t.Error("failed")
	}
	v = cmdconfig.GetStringSlice("aaa")
	if v != nil {
		t.Errorf("failed: %++v\n", v)
	}
}

func Test_Int(t *testing.T) {
	v := cmdconfig.GetInt("int")
	if v != 1 {
		t.Error("failed")
	}
	if cmdconfig.GetInt("aaa") != 0 {
		t.Error("failed")
	}
}

func Test_GetBool(t *testing.T) {
	v := cmdconfig.GetBool("bool")
	if !v {
		t.Error("failed")
	}
	if cmdconfig.GetBool("aaa") {
		t.Error("failed")
	}
}

func Test_IsSet(t *testing.T) {
	cmdconfig.Set("key", "value")
	if !cmdconfig.IsSet("key") {
		t.Error("failed")
	}
	if cmdconfig.IsSet("aaa") {
		t.Error("failed")
	}
}
