package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

type addr struct {
	listener net.Listener
	value    string
	insecure bool
}

func (a *addr) Set(value string) error {
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
		return errors.Tracef(err)
	}

	port = strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)

	a.value = fmt.Sprintf("%v:%v", host, port)
	a.listener = listener

	return nil
}

func (a *addr) Listener() (net.Listener, error) {
	if a.listener == nil {
		err := a.Set(":0")
		if err != nil {
			return nil, errors.Tracef(err)
		}
	}

	return a.listener, nil
}

func (a addr) String() string {
	protocol := "https"
	if a.insecure {
		protocol = "http"
	}

	return fmt.Sprintf("%v://%v", protocol, a.value)
}
