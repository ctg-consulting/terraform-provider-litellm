package provider

import (
	"net/http"
	"time"

	"cloud.google.com/go/auth"
	"cloud.google.com/go/auth/credentials"
	"cloud.google.com/go/auth/credentials/idtoken"
	"cloud.google.com/go/auth/httptransport"
)

func newCloudRunCredentials(audience string, newClient func(*httptransport.Options) (*http.Client, error)) (*auth.Credentials, error) {
	// Impersonation requires an authenticated client for the IAM ID-token request.
	client, err := newClient(&httptransport.Options{
		DetectOpts: &credentials.DetectOptions{Scopes: []string{"https://www.googleapis.com/auth/cloud-platform"}},
	})
	if err != nil {
		return nil, err
	}
	client.Timeout = 30 * time.Second
	return idtoken.NewCredentials(&idtoken.Options{Audience: audience, Client: client})
}
