package main

import (
	"encoding/json"
	"strings"
)

type Proxies []string

func (p *Proxies) Set(value string) error {
	if value == "" {
		return nil
	}

	*p = strings.Fields(value)

	return nil
}

func (p Proxies) String() string {
	if p == nil {
		return ""
	}

	b, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return string(b)
}
