package sms

import (
	"context"
	"errors"
)

var ErrInvalidNumber = errors.New("invalid number")

type Messager interface {
	SendSMS(ctx context.Context, from, to, body string) error
}
