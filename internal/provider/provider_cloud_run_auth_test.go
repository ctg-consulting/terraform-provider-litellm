package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/auth/httptransport"
)

func TestProviderCloudRunWIFAuthenticatesIDTokenRequest(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	subjectFile := filepath.Join(dir, "subject")
	if err := os.WriteFile(subjectFile, []byte("subject-token"), 0600); err != nil {
		t.Fatal(err)
	}
	adc, err := json.Marshal(map[string]any{
		"type": "external_account", "audience": "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/test/providers/test",
		"subject_token_type": "urn:ietf:params:oauth:token-type:jwt", "token_url": "https://sts.googleapis.com/v1/token",
		"service_account_impersonation_url": "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/runner@test.iam.gserviceaccount.com:generateAccessToken",
		"credential_source":                 map[string]any{"file": subjectFile},
	})
	if err != nil {
		t.Fatal(err)
	}
	adcFile := filepath.Join(dir, "adc.json")
	if err := os.WriteFile(adcFile, adc, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", adcFile)
	t.Setenv("GOOGLE_API_USE_CLIENT_CERTIFICATE", "false")
	audience := "https://gateway.run.app"
	payload := fmt.Sprintf(`{"exp":%d,"aud":%q}`, time.Now().Add(time.Hour).Unix(), audience)
	idToken := "eyJhbGciOiJSUzI1NiJ9." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".c2ln"
	calls := map[string]int{}
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body string
		switch {
		case r.URL.Host == "sts.googleapis.com":
			calls["sts"]++
			body = `{"access_token":"sts-access","token_type":"Bearer","expires_in":3600}`
		case strings.HasSuffix(r.URL.Path, ":generateAccessToken"):
			calls["access"]++
			if r.Header.Get("Authorization") != "Bearer sts-access" {
				return nil, fmt.Errorf("missing STS authorization")
			}
			body = fmt.Sprintf(`{"accessToken":"sa-access","expireTime":%q}`, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
		case strings.HasSuffix(r.URL.Path, ":generateIdToken"):
			calls["id"]++
			if r.Header.Get("Authorization") != "Bearer sa-access" {
				return nil, fmt.Errorf("missing service-account authorization")
			}
			var input struct {
				Audience     string `json:"audience"`
				IncludeEmail bool   `json:"includeEmail"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				return nil, err
			}
			if input.Audience != audience || !input.IncludeEmail {
				return nil, fmt.Errorf("incorrect ID-token claims")
			}
			body = fmt.Sprintf(`{"token":%q}`, idToken)
		default:
			return nil, fmt.Errorf("unexpected authentication request")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	creds, err := newCloudRunCredentials(audience, func(options *httptransport.Options) (*http.Client, error) {
		options.BaseRoundTripper = http.DefaultTransport
		return httptransport.NewClient(options)
	})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		token, err := creds.Token(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if token.Value != idToken {
			t.Fatal("unexpected ID token")
		}
	}
	if calls["sts"] != 1 || calls["access"] != 1 || calls["id"] != 1 {
		t.Fatalf("unexpected authentication calls: %v", calls)
	}
}
