package main

import (
	"encoding/json"
	"strings"
)

type proxies []string

func (p *proxies) Set(value string) error {
	if value == "" {
		return nil
	}

	*p = strings.Fields(value)

	return nil
}

func (p proxies) String() string {
	if p == nil {
		return ""
	}

	b, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return string(b)
}
