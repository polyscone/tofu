package realip

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
)

var ErrTooManyAddresses = errors.New("too many addresses")

// FromRequest extracts the real ip address from the request parameters.
//
// If no trusted proxy addresses are given then the result will always be the
// request's remote address.
//
// In the case where a list of trusted proxies is given then the address to the
// left of the rightmost address in the x-forwarded-for chain is returned
// assuming the remote address is also a trusted proxy.
func FromRequest(r *http.Request, proxies []string) (string, error) {
	remoteAddr := r.RemoteAddr
	if strings.Contains(remoteAddr, ":") {
		ip, _, err := net.SplitHostPort(remoteAddr)
		if err != nil {
			return "", fmt.Errorf("split host port: %w", err)
		}

		remoteAddr = ip
	}

	xff := r.Header.Values("x-forwarded-for")
	if proxies == nil || xff == nil {
		return remoteAddr, nil
	}

	// Each string may contain multiple addresses separated by commas, for example...
	// 	[]string{
	// 		"1.1.1.1",
	// 		"2.2.2.2, 3.3.3.3",
	// 		"4.4.4.4",
	// 	}
	// ...so we join all of them together with commas and split them by
	// comma again to ensure we have one address per string
	all := strings.Join(xff, ",") + "," + remoteAddr
	addrs := strings.Split(all, ",")

	const max = 50
	if len(addrs) > max {
		return "", ErrTooManyAddresses
	}

	for i := len(addrs) - 1; i >= 0; i-- {
		addr := strings.TrimSpace(addrs[i])

		if !slices.Contains(proxies, addr) {
			return addr, nil
		}
	}

	return addrs[0], nil
}
