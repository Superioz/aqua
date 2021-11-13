package env

import (
	"os"
	"strconv"
)

func String(name string) (string, bool) {
	return os.LookupEnv(name)
}

func StringOrDefault(name string, def string) string {
	s, ok := String(name)
	if !ok {
		return def
	}
	return s
}

func Int(name string) (int, bool) {
	s, ok := String(name)
	if !ok {
		return 0, false
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return i, true
}

func IntOrDefault(name string, def int) int {
	i, ok := Int(name)
	if !ok {
		return def
	}
	return i
}

func Bool(name string) (bool, bool) {
	s, ok := String(name)
	if !ok {
		return false, false
	}

	if s == "true" || s == "enabled" {
		return true, true
	}
	return false, true
}

func BoolOrDefault(name string, def bool) bool {
	b, ok := Bool(name)
	if !ok {
		return def
	}
	return b
}
