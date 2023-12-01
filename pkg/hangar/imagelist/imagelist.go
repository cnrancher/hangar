package imagelist

import "strings"

type ListType string

const (
	// TypeUnknow is the undefined format.
	TypeUnknow ListType = ""

	// TypeMirror:
	//
	//  [SOURCE_IMAGE] [DEST_IMAGE] [TAG]
	//
	// Example:
	//  docker.io/library/mysql docker.io/username/mirrored-mysql latest
	//  quay.io/skopeo/stable docker.io/username/mirrored-skopeo-stable 1.22
	TypeMirror ListType = "mirror"

	// TypeDefault:
	//
	//  [REGISTRY]/[PROJECT]/[NAME]:[TAG]
	//
	// Example:
	//  docker.io/library/nginx:1.22
	TypeDefault ListType = "default"
)

func IsMirrorFormat(line string) bool {
	_, ok := getMirrorSpec(line)
	return ok
}

func GetMirrorSpec(line string) ([]string, bool) {
	return getMirrorSpec(line)
}

func IsDefaultFormat(line string) bool {
	return isDefaultFormat(line)
}

func Detect(line string) ListType {
	_, ok := getMirrorSpec(line)
	if ok {
		return TypeMirror
	}
	if isDefaultFormat(line) {
		return TypeDefault
	}
	return TypeUnknow
}

func getMirrorSpec(line string) ([]string, bool) {
	line = strings.TrimSpace(line)
	v := strings.Split(line, " ")
	var spec = make([]string, 0, 3)
	for _, s := range v {
		if len(s) == 0 {
			continue
		}
		spec = append(spec, s)
	}
	if len(spec) != 3 {
		return nil, false
	}
	return spec, true
}

func isDefaultFormat(line string) bool {
	line = strings.TrimSpace(line)
	v := strings.Split(line, "/")
	var spec = make([]string, 0, 3)
	for _, s := range v {
		if len(s) == 0 {
			continue
		}
		if strings.Contains(s, " ") {
			continue
		}
		spec = append(spec, s)
	}
	if len(spec) < 1 || len(spec) > 3 {
		return false
	}
	return true
}
