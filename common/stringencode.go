package common

import (
	"fmt"
	"sort"
	"strings"
)

func SortAndEncodeMap(data map[string]string) string {
	mapkeys := make([]string, 0)
	for k, _ := range data {
		mapkeys = append(mapkeys, k)
	}
	sort.Strings(mapkeys)
	strbuf := ""
	for i, key := range mapkeys {
		val := data[key]
		if val == "" {
			continue
		}
		strbuf += fmt.Sprintf("%s=%s", key, val)
		if i < len(mapkeys)-1 {
			strbuf += "&"
		}
	}
	return strbuf
}

func StringDecodeMap(s string) map[string]string {
	if s == "" {
		return nil
	}
	kvs := strings.Split(s, "&")
	result := make(map[string]string)
	for _, kv := range kvs {
		mkv := strings.Split(kv, "=")
		if len(mkv) < 2 {
			continue
		}
		key, value := mkv[0], mkv[1]
		result[key] = value
	}
	return result
}
