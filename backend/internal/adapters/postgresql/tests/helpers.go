package db_tests

import (
	"strings"
)

func splitEnumName(fullName string) (string, string) {
	parts := strings.Split(fullName, ".")
	var schema, typeName string

	if len(parts) == 2 {
		schema = parts[0]
		typeName = parts[1]
	} else {
		schema = "public"
		typeName = parts[0]
	}

	return schema, typeName
}
