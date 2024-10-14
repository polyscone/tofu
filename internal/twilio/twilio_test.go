package twilio_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/twilio"
)

func TestMessages(t *testing.T) {
	tt := []struct {
		name    string
		sid     string
		token   string
		from    string
		to      string
		body    string
		wantErr bool
	}{
		{"success", "AC0123456789abcdef0123456789abcdef", "token123", "+818000000001", "+818000000002", "Test Body", false},
		{"success with longest body", "AC0123456789abcdef0123456789abcdef", "token123", "+818000000001", "+818000000002", strings.Repeat("X", 1600), false},

		{"invalid empty sid", "", "token123", "+818000000001", "+818000000002", "Test Body", true},
		{"invalid empty sid with spaces", "      ", "token123", "+818000000001", "+818000000002", "Test Body", true},
		{"invalid sid format prefix", "AB0123456789abcdef0123456789abcdef", "token123", "+818000000001", "+818000000002", "Test Body", true},
		{"invalid sid format length", "ACXX", "token123", "+818000000001", "+818000000002", "Test Body", true},

		{"invalid empty token", "AC0123456789abcdef0123456789abcdef", "", "+818000000001", "+818000000002", "Test Body", true},
		{"invalid empty token with spaces", "AC0123456789abcdef0123456789abcdef", "    ", "+818000000001", "+818000000002", "Test Body", true},

		{"invalid empty to", "AC0123456789abcdef0123456789abcdef", "token123", "+818000000001", "", "Test Body", true},
		{"invalid empty to with spaces", "AC0123456789abcdef0123456789abcdef", "token123", "+818000000001", "     ", "Test Body", true},
		{"invalid to without + prefix", "AC0123456789abcdef0123456789abcdef", "token123", "+818000000001", "818000000002", "Test Body", true},

		{"invalid empty from", "AC0123456789abcdef0123456789abcdef", "token123", "", "+818000000002", "Test Body", true},
		{"invalid empty from with spaces", "AC0123456789abcdef0123456789abcdef", "token123", "     ", "+818000000002", "Test Body", true},
		{"invalid from without + prefix", "AC0123456789abcdef0123456789abcdef", "token123", "818000000001", "+818000000002", "Test Body", true},

		{"invalid empty body", "AC0123456789abcdef0123456789abcdef", "token123", "+818000000001", "+818000000002", "", true},
		{"invalid empty body with spaces", "AC0123456789abcdef0123456789abcdef", "token123", "+818000000001", "+818000000002", "      ", true},
		{"invalid body too long", "AC0123456789abcdef0123456789abcdef", "token123", "+818000000001", "+818000000002", strings.Repeat("X", 1601), true},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var callCount int
			wantCalls := 1
			wantMethod := http.MethodPost
			wantRequestURI := "/2010-04-01/Accounts/" + tc.sid + "/Messages.json"
			wantContentType := "application/x-www-form-urlencoded"
			wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(tc.sid+":"+tc.token))

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++

				if m := r.Method; m != wantMethod {
					t.Errorf("want method %q, got %q", wantMethod, m)
				}
				if p := r.URL.RequestURI(); p != wantRequestURI {
					t.Errorf("want request uri %q, got %q", wantRequestURI, p)
				}
				if h := r.Header.Get("content-type"); h != wantContentType {
					t.Errorf("want content-type header %q, got %q", wantContentType, h)
				}
				if h := r.Header.Get("authorization"); h != wantAuth {
					t.Errorf("want authorization header %q, got %q", wantAuth, h)
				}
				if s := r.PostFormValue("To"); s != tc.to {
					t.Errorf("want to value %q, got %q", tc.to, s)
				}
				if s := r.PostFormValue("From"); s != tc.from {
					t.Errorf("want from value %q, got %q", tc.from, s)
				}
				if s := r.PostFormValue("Body"); s != tc.body {
					t.Errorf("want body value %q, got %q", tc.body, s)
				}

				data := map[string]any{
					"account_sid":  tc.sid,
					"api_version":  "2010-04-01",
					"body":         tc.body,
					"date_created": "Thu, 30 Jul 2015 20:12:31 +0000",
					"date_sent":    "Thu, 30 Jul 2015 20:12:33 +0000",
					"date_updated": "Thu, 30 Jul 2015 20:12:33 +0000",
					"direction":    "outbound-api",
					"from":         tc.from,
					"num_media":    "0",
					"num_segments": "1",
					"price":        -0.00750,
					"price_unit":   "USD",
					"status":       "sent",
					"to":           tc.to,
				}
				if err := json.NewEncoder(w).Encode(data); err != nil {
					t.Fatal(err)
				}
			}))
			defer ts.Close()

			client := http.Client{Timeout: 10 * time.Second}
			tw := twilio.NewTwilioClient(&client, tc.sid, tc.token)
			tw.Endpoint = ts.URL

			err := tw.SendSMS(context.Background(), tc.from, tc.to, tc.body)
			if tc.wantErr {
				if err == nil {
					t.Error("want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			if callCount != wantCalls {
				t.Errorf("want %d post requests, got %d", wantCalls, callCount)
			}
		})
	}
}
