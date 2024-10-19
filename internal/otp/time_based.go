package otp

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

var ErrPasswordUsed = errors.New("the provided password has already been used")

var usedTOTPs = struct {
	sync.RWMutex
	data map[string]time.Time
}{data: make(map[string]time.Time)}

// CleanUsedTOTP will clear any recorded used TOTP that matches
// the given string.
func CleanUsedTOTP(password string) {
	usedTOTPs.Lock()
	defer usedTOTPs.Unlock()

	delete(usedTOTPs.data, password)
}

// CleanUsedTOTPs will clear all recorded used TOTPs that are older than the
// given age duration.
func CleanUsedTOTPs(age time.Duration) {
	usedTOTPs.Lock()
	defer usedTOTPs.Unlock()

	for password, t := range usedTOTPs.data {
		if time.Since(t) > age {
			delete(usedTOTPs.data, password)
		}
	}
}

// TimeBased implements a "Time-based One Time Password" (TOTP) in accordance
// with RFC6238 and any errata.
type TimeBased struct {
	hmacBased HMACBased
	baseTime  time.Time
	timeStep  time.Duration
}

func NewTimeBased(digits int, alg Algorithm, baseTime time.Time, timeStep time.Duration) (TimeBased, error) {
	otp := TimeBased{
		baseTime: baseTime,
		timeStep: timeStep,
	}

	hmacBased, err := NewHMACBased(digits, alg)
	if err != nil {
		return otp, fmt.Errorf("new HMAC based: %w", err)
	}

	otp.hmacBased = hmacBased

	if otp.baseTime.IsZero() || otp.baseTime.Unix() < 0 {
		return otp, errors.New("base time must be set to at least the unix epoch (0)")
	}
	if otp.timeStep.Seconds() < 30.0 {
		return otp, errors.New("time step must be at least 30 seconds")
	}

	return otp, nil
}

func (otp TimeBased) Generate(key []byte, t time.Time) (string, error) {
	if t.IsZero() || t.Unix() < 0 {
		return "", errors.New("time must be set to at least the unix epoch (0)")
	}

	count := uint64(math.Floor(float64(t.Unix()-otp.baseTime.Unix()) / otp.timeStep.Seconds()))
	totp, err := otp.hmacBased.Generate(key, count)
	if err != nil {
		return "", err
	}

	return totp, nil
}

func (otp TimeBased) Check(key []byte, t time.Time, delaySteps int, userPassword string) (bool, error) {
	if delaySteps < 0 {
		return false, errors.New("delay steps cannot be negative")
	}

	if wantMax := 4; delaySteps > wantMax {
		return false, fmt.Errorf("delay steps is too large, a maximum of %v is expected", wantMax)
	}

	if len(key) == 0 || len(userPassword) == 0 {
		return false, nil
	}

	defer CleanUsedTOTPs(10 * time.Minute)

	usedTOTPs.RLock()
	if _, ok := usedTOTPs.data[userPassword]; ok {
		usedTOTPs.RUnlock()

		return false, ErrPasswordUsed
	}
	usedTOTPs.RUnlock()

	for i := -delaySteps; i <= delaySteps; i++ {
		offset := otp.timeStep * time.Duration(i)
		password, err := otp.Generate(key, t.Add(offset))
		if err != nil {
			return false, fmt.Errorf("generate: %w", err)
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
