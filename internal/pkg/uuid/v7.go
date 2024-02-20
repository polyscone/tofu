package uuid

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"regexp"
	"sync"
	"time"
)

type sequenceClock struct {
	mu   sync.Mutex
	last int64
	seq  uint16
}

func newSequenceClock() *sequenceClock {
	var s sequenceClock

	s.init()

	return &s
}

func (s *sequenceClock) init() {
	b := make([]byte, 2)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(fmt.Sprintf("read random bytes: %v", err))
	}

	// The most significant 4 bits will be replaced, so we don't need those
	// and we want to leave some room for the sequence to increase, so we
	// discard the 12th bit as well, leaving us with 11 random bits as a
	// sequence starting point
	s.seq = binary.BigEndian.Uint16(b) & 0x7FF
}

func (s *sequenceClock) now() (int64, uint16, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()

	// Ensure the now value is actually greater than the last one
	for now < s.last {
		time.Sleep(100 * time.Nanosecond)

		now = time.Now().UnixMilli()
	}

	// Handle duplicate timestamps by incrementing a sequence number
	// or waiting until the next millisecond timestamp
	if now == s.last {
		if s.seq < 0xFFF {
			// If the sequence number would fit in 12 bits we'll increment it
			s.seq++
		} else {
			// If the sequence number would overflow 12 bits
			// we'll wait until the next millisecond timestamp and
			// reinitialise the sequence number
			for now <= s.last {
				time.Sleep(100 * time.Nanosecond)

				now = time.Now().UnixMilli()
			}

			s.init()
		}
	}

	s.last = now

	return now, s.seq, nil
}

var v7clock = newSequenceClock()

var validV7 = regexp.MustCompile("(?i)^[0-9A-F]{8}-[0-9A-F]{4}-7[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$")

func NewV7() (UUID, error) {
	var id UUID

	// ms is the unix time in milliseconds
	// seq is a number that's incremented for duplicate ms values to support batches
	ms, seq, err := v7clock.now()
	if err != nil {
		return Nil, err
	}

	// Unix millisecond timestamp in big-endian
	id[0] = byte(ms >> 40)
	id[1] = byte(ms >> 32)
	id[2] = byte(ms >> 24)
	id[3] = byte(ms >> 16)
	id[4] = byte(ms >> 8)
	id[5] = byte(ms >> 0)

	// The sequence number covers 2 bytes but the most
	// significant 4 will be replaced with the version number
	binary.BigEndian.PutUint16(id[6:8], seq)

	if _, err := io.ReadFull(rand.Reader, id[8:16]); err != nil {
		return Nil, fmt.Errorf("read random bytes: %w", err)
	}

	id[6] = (id[6] & 0x0F) | (0x07 << 4)    // Set version to 7
	id[8] = (id[8]&(0xFF>>2) | (0x02 << 6)) // Set variant to RFC4122

	return id, nil
}

func (id UUID) IsValidV7() bool {
	return validV7.MatchString(id.String())
}
