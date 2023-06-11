package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Addr struct {
	listener net.Listener
	value    string
	insecure bool
}

func (a *Addr) Set(value string) error {
	if value == "" {
		return nil
	}

	host, port, _ := strings.Cut(value, ":")
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "0"
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen on %v: %w", port, err)
	}

	port = strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)

	a.value = fmt.Sprintf("%v:%v", host, port)
	a.listener = listener

	return nil
}

func (a *Addr) Listener() (net.Listener, error) {
	if a.listener == nil {
		err := a.Set(":0")
		if err != nil {
			return nil, err
		}
	}

	return a.listener, nil
}

func (a Addr) String() string {
	protocol := "https"
	if a.insecure {
		protocol = "http"
	}

	return fmt.Sprintf("%v://%v", protocol, a.value)
}
