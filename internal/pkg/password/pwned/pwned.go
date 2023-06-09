package pwned

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

var client = http.Client{Timeout: 10 * time.Second}

var ErrEmptyPassword = errors.New("no password was provided")

func KnownPasswordBreachCount(ctx context.Context, password []byte) (int, error) {
	if len(bytes.TrimSpace(password)) == 0 {
		return 0, errors.Tracef(ErrEmptyPassword)
	}

	h := sha1.New()
	if _, err := h.Write(password); err != nil {
		return 0, errors.Tracef(err)
	}
	encoded := strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
	needlePrefix := encoded[:5]
	needleSuffix := encoded[5:]

	endpoint := "https://api.pwnedpasswords.com/range/" + needlePrefix
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, errors.Tracef(err)
	}

	res, err := client.Do(req)
	if err != nil {
		return 0, errors.Tracef(err)
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, errors.Tracef(err)
	}
	results := string(b)

	for _, candidate := range strings.Split(results, "\n") {
		suffix, count, _ := strings.Cut(strings.TrimSpace(candidate), ":")
		if suffix != needleSuffix {
			continue
		}

		n, err := strconv.Atoi(count)
		if err != nil {
			return 0, errors.Tracef(err)
		}

		return n, nil
	}

	return 0, nil
}
