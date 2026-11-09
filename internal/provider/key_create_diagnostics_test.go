package provider

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestGeneratedKeyCreateDiagnosticsRemainContentFree(t *testing.T) {
	t.Parallel()
	secret := "sk-private-key-value-never-log"
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"api", &APIError{StatusCode: http.StatusBadRequest, Detail: secret, Body: secret}, "HTTP status: 400"},
		{"before-dispatch", safeTransportFailure(errors.New(secret)), "request dispatch confirmed: false"},
		{"after-dispatch", safeDispatchedTransportFailure(errors.New(secret)), "request dispatch confirmed: true"},
		{"accepted-response", &safeResponseError{statusCode: 200, kind: secret, requestID: secret, dispatched: true, accepted: true}, "response acceptance confirmed: true"},
		{"opaque", errors.New(secret), "HTTP status: unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message := generatedKeyCreateError(test.err)
			if strings.Contains(message, secret) || !strings.Contains(message, test.want) {
				t.Fatalf("unexpected content-free diagnostic: %q", message)
			}
		})
	}
}
