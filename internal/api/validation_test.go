package api

import (
	"testing"
)

func TestValidateURL(t *testing.T) {
	type testCase struct {
		name    string
		url     string
		wantErr bool
	}

	cases := []testCase{
		// valid URLs
		{"valid https", "https://example.com", false},
		{"valid http", "http://example.com", false},

		// invalid formats
		{"empty url", "", true},
		{"no scheme", "example.com", true},
		{"no host", "http://", true},

		// unsupported scheme
		{"ftp scheme", "ftp://example.com", true},

		// blocked IP ranges
		{"loopback", "http://127.0.0.1", true},
		{"private 192", "http://192.168.1.1", true},
		{"private 10", "http://10.0.0.1", true},
		{"private 172", "http://172.16.0.1", true},
		{"link local", "http://169.254.169.254", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateURL(tc.url)

			// determine if the outcome is what we expected
			// err!=nil is either true or false
			if (err != nil) != tc.wantErr {
				t.Errorf("validateURL(%q) error = %v, wantErr %v", tc.url, err, tc.wantErr)
			}
		})
	}
}
