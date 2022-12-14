package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

type Addr struct {
	Value    string
	Insecure bool
}

func (a *Addr) Set(value string) error {
	if value == "" {
		return nil
	}

	host, port, _ := strings.Cut(value, ":")
	if host == "" {
		host = "localhost"
	}
	if port == "" || port == "0" {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return errors.Tracef(err)
		}

		port = strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	}

	a.Value = fmt.Sprintf("%v:%v", host, port)

	return nil
}

func (a Addr) String() string {
	protocol := "https"
	if a.Insecure {
		protocol = "http"
	}

	return fmt.Sprintf("%v://%v", protocol, a.Value)
}
