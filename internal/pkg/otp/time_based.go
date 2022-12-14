package otp

import (
	"crypto/subtle"
	"math"
	"sync"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

var ErrPasswordUsed = errors.New("the provided password has already been used")

var usedTOTPs = struct {
	sync.RWMutex
	data map[string]time.Time
}{data: make(map[string]time.Time)}

// TimeBased implements a "Time-based One Time Password" (TOTP) in accordance
// with RFC6238 and any errata.
type TimeBased struct {
	hmacBased HMACBased
	baseTime  time.Time
	timeStep  time.Duration
}

func NewTimeBased(digits int, alg Alg, baseTime time.Time, timeStep time.Duration) (TimeBased, error) {
	otp := TimeBased{
		baseTime: baseTime,
		timeStep: timeStep,
	}

	hmacBased, err := NewHMACBased(digits, alg)
	if err != nil {
		return otp, err
	}

	otp.hmacBased = hmacBased

	if otp.baseTime.IsZero() || otp.baseTime.Unix() < 0 {
		return otp, errors.Tracef("base time must be set to at least the unix epoch (0)")
	}
	if otp.timeStep.Seconds() < 30.0 {
		return otp, errors.Tracef("time step must be at least 30 seconds")
	}

	return otp, nil
}

func (otp TimeBased) Generate(key []byte, t time.Time) (string, error) {
	if t.IsZero() || t.Unix() < 0 {
		return "", errors.Tracef("time must be set to at least the unix epoch (0)")
	}

	count := uint64(math.Floor(float64(t.Unix()-otp.baseTime.Unix()) / otp.timeStep.Seconds()))
	totp, err := otp.hmacBased.Generate(key, count)
	if err != nil {
		return "", err
	}

	return totp, nil
}

func (otp TimeBased) Verify(key []byte, t time.Time, delaySteps int, userPassword string) (bool, error) {
	if delaySteps < 0 {
		return false, errors.Tracef("delay steps cannot be negative")
	}

	if wantMax := 4; delaySteps > wantMax {
		return false, errors.Tracef("delay steps is too large, a maximum of %v is expected", wantMax)
	}

	if len(key) == 0 || len(userPassword) == 0 {
		return false, nil
	}

	// Defer a function to do a clean up of used OTPs in memory
	// after each verification
	defer func() {
		usedTOTPs.Lock()
		defer usedTOTPs.Unlock()

		for password, t := range usedTOTPs.data {
			if time.Since(t) > 10*time.Minute {
				delete(usedTOTPs.data, password)
			}
		}
	}()

	usedTOTPs.RLock()
	if _, ok := usedTOTPs.data[userPassword]; ok {
		usedTOTPs.RUnlock()

		return false, ErrPasswordUsed
	}
	usedTOTPs.RUnlock()

	// Check into the past
	for i := 0; i <= int(delaySteps); i++ {
		step := otp.timeStep * time.Duration(i)
		password, err := otp.Generate(key, t.Add(-step))
		if err != nil {
			return false, err
		}

		if subtle.ConstantTimeCompare([]byte(password), []byte(userPassword)) == 1 {
			usedTOTPs.Lock()
			usedTOTPs.data[password] = time.Now()
			usedTOTPs.Unlock()

			return true, nil
		}
	}

	// Check into the future
	// We start at index 1 here because the checks into the past already include
	// the current time (i = 0)
	for i := 1; i <= int(delaySteps); i++ {
		step := otp.timeStep * time.Duration(i)
		password, err := otp.Generate(key, t.Add(step))
		if err != nil {
			return false, err
		}

		if subtle.ConstantTimeCompare([]byte(password), []byte(userPassword)) == 1 {
			usedTOTPs.Lock()
			usedTOTPs.data[password] = time.Now()
			usedTOTPs.Unlock()

			return true, nil
		}
	}

	return false, nil
}
