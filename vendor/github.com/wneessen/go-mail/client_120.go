// SPDX-FileCopyrightText: 2022-2023 The go-mail Authors
//
// SPDX-License-Identifier: MIT

//go:build go1.20
// +build go1.20

package mail

import (
	"errors"
	"strings"
)

// Send sends out the mail message
func (c *Client) Send(ml ...*Msg) (rerr error) {
	if err := c.checkConn(); err != nil {
		rerr = &SendError{Reason: ErrConnCheck, errlist: []error{err}, isTemp: isTempError(err)}
		return
	}
	for _, m := range ml {
		m.sendError = nil
		if m.encoding == NoEncoding {
			if ok, _ := c.sc.Extension("8BITMIME"); !ok {
				m.sendError = &SendError{Reason: ErrNoUnencoded, isTemp: false}
				rerr = errors.Join(rerr, m.sendError)
				continue
			}
		}
		f, err := m.GetSender(false)
		if err != nil {
			m.sendError = &SendError{Reason: ErrGetSender, errlist: []error{err}, isTemp: isTempError(err)}
			rerr = errors.Join(rerr, m.sendError)
			continue
		}
		rl, err := m.GetRecipients()
		if err != nil {
			m.sendError = &SendError{Reason: ErrGetRcpts, errlist: []error{err}, isTemp: isTempError(err)}
			rerr = errors.Join(rerr, m.sendError)
			continue
		}

		if c.dsn {
			if c.dsnmrtype != "" {
				c.sc.SetDSNMailReturnOption(string(c.dsnmrtype))
			}
		}
		if err := c.sc.Mail(f); err != nil {
			m.sendError = &SendError{Reason: ErrSMTPMailFrom, errlist: []error{err}, isTemp: isTempError(err)}
			rerr = errors.Join(rerr, m.sendError)
			if reserr := c.sc.Reset(); reserr != nil {
				rerr = errors.Join(rerr, reserr)
			}
			continue
		}
		failed := false
		rse := &SendError{}
		rse.errlist = make([]error, 0)
		rse.rcpt = make([]string, 0)
		rno := strings.Join(c.dsnrntype, ",")
		c.sc.SetDSNRcptNotifyOption(rno)
		for _, r := range rl {
			if err := c.sc.Rcpt(r); err != nil {
				rse.Reason = ErrSMTPRcptTo
				rse.errlist = append(rse.errlist, err)
				rse.rcpt = append(rse.rcpt, r)
				rse.isTemp = isTempError(err)
				failed = true
			}
		}
		if failed {
			if reserr := c.sc.Reset(); reserr != nil {
				rerr = errors.Join(rerr, reserr)
			}
			m.sendError = rse
			rerr = errors.Join(rerr, m.sendError)
			continue
		}
		w, err := c.sc.Data()
		if err != nil {
			m.sendError = &SendError{Reason: ErrSMTPData, errlist: []error{err}, isTemp: isTempError(err)}
			rerr = errors.Join(rerr, m.sendError)
			continue
		}
		_, err = m.WriteTo(w)
		if err != nil {
			m.sendError = &SendError{Reason: ErrWriteContent, errlist: []error{err}, isTemp: isTempError(err)}
			rerr = errors.Join(rerr, m.sendError)
			continue
		}
		m.isDelivered = true

		if err := w.Close(); err != nil {
			m.sendError = &SendError{Reason: ErrSMTPDataClose, errlist: []error{err}, isTemp: isTempError(err)}
			rerr = errors.Join(rerr, m.sendError)
			continue
		}

		if err := c.Reset(); err != nil {
			m.sendError = &SendError{Reason: ErrSMTPReset, errlist: []error{err}, isTemp: isTempError(err)}
			rerr = errors.Join(rerr, m.sendError)
			continue
		}
		if err := c.checkConn(); err != nil {
			m.sendError = &SendError{Reason: ErrConnCheck, errlist: []error{err}, isTemp: isTempError(err)}
			rerr = errors.Join(rerr, m.sendError)
		}
	}

	return
}
