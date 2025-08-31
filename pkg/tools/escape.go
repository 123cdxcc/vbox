package tools

import "strings"

func EscapeDockerName(name string) string {
	return strings.TrimPrefix(name, "/")
}
