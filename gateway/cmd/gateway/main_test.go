package main

import "testing"

func TestInvalidRequestTimeout(t *testing.T) {
	for _, value := range []string{"bad", "0", "-1s"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("HTTP_REQUEST_TIMEOUT", value)
			if err := run(); err == nil || err.Error() != "HTTP_REQUEST_TIMEOUT must be a positive duration" {
				t.Fatalf("invalid timeout: %v", err)
			}
		})
	}
}
