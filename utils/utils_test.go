package utils

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

const (
	exampleJson = `
{
	"example_string": "abc",
	"example_float": 1.23,
	"example_int": 123,
	"example_nested_obj": {
		"a": "a"
	},
	"example_array": [
		{"a": "a"},{ "b": "b"}
	]
}`
)

func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestSha256(t *testing.T) {
	s := Sha256Sum("123")
	if s != "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3" {
		t.Errorf("sha256 test failed")
	}
	s = Sha256Sum("<nil>")
	if s != "a9dc16a7a3875d174c1af4f923f261cafc124357f2322493a59ee0d14fcd10db" {
		t.Errorf("sha256 test failed")
	}
}

func TestJson(t *testing.T) {
	var testJson map[string]interface{}
	err := json.NewDecoder(strings.NewReader(exampleJson)).Decode(&testJson)
	if err != nil {
		t.Error(err.Error())
		return
	}
	if f, ok := ReadJsonFloat64(testJson, "example_float"); !ok || !floatEqual(f, 1.23) {
		t.Error("ReadJsonFloat64 failed")
	}
	if v, ok := ReadJsonInt(testJson, "example_int"); !ok || v != 123 {
		t.Error("ReadJsonInt failed")
	}
	if v, ok := ReadJsonString(testJson, "example_string"); !ok || v != "abc" {
		t.Error("ReadJsonString failed")
	}
	if _, ok := ReadJsonSubObj(testJson, "example_nested_obj"); !ok {
		t.Error("ReadJsonSubObj failed")
	}
	if _, ok := ReadJsonSubArray(testJson, "example_array"); !ok {
		t.Error("ReadJsonSubArray failed")
	}
}
