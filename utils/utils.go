package utils

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

var (
	ErrReadJsonFailed   = errors.New("failed to read value from json")
	ErrSkopeoNotFound   = errors.New("skopeo not found")
	ErrDockerNotFound   = errors.New("docker not found")
	ErrLoginFailed      = errors.New("login failed")
	ErrNoAvailableImage = errors.New("no image available for specified arch list")
)

const (
	DockerLoginURL    = "https://hub.docker.com/v2/users/login/"
	DockerHubRegistry = "docker.io"
)

func ReadJsonStringVal(j map[string]interface{}, k string) (string, bool) {
	v, ok := j[k]
	if !ok {
		return "", false
	}
	return v.(string), true
}

func ReadJsonFloat64Val(j map[string]interface{}, k string) (float64, bool) {
	v, ok := j[k]
	if !ok {
		return 0, false
	}
	return v.(float64), true
}

func ReadJsonIntVal(j map[string]interface{}, k string) (int, bool) {
	v, ok := j[k]
	if !ok {
		return 0, false
	}
	return int(v.(float64)), true
}

func ReadJsonSubObj(j map[string]interface{}, k string) (map[string]interface{}, bool) {
	v, ok := j[k]
	if !ok {
		return nil, false
	}
	return v.(map[string]interface{}), true
}

func ReadJsonSubArray(j map[string]interface{}, k string) ([]interface{}, bool) {
	v, ok := j[k]
	if !ok {
		return nil, false
	}
	return v.([]interface{}), true
}

func Sha256Sum(s string) string {
	sum := sha256.Sum256([]byte("hello world\n"))
	return fmt.Sprintf("%x", sum)
}
