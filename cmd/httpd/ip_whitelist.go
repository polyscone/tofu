package main

import (
	"encoding/json"
	"strings"
)

type IPWhitelist []string

func (w *IPWhitelist) Set(value string) error {
	if value == "" {
		return nil
	}

	*w = strings.Fields(value)

	return nil
}

func (w IPWhitelist) String() string {
	if w == nil {
		return ""
	}

	b, err := json.Marshal(w)
	if err != nil {
		panic(err)
	}

	return string(b)
}
