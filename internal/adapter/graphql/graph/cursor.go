package graph

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

func encodeCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("cursor:%d", offset)))
}

func decodeCursor(cursor string) int {
	b, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0
	}
	s := string(b)
	if !strings.HasPrefix(s, "cursor:") {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimPrefix(s, "cursor:"))
	if err != nil {
		return 0
	}
	return n
}
