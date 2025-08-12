package utils

import (
	"os"
	"strconv"
)

func GetEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func GetEnvInt(key string, defaultValue int) int {
	if valstr := os.Getenv(key); valstr != "" {
		val, err := strconv.Atoi(valstr)
		if err != nil {
			return defaultValue
		}
		return val
	}
	return defaultValue
}
