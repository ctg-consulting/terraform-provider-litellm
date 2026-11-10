package provider

import (
	"context"
	"errors"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCloudRunHTTPClientWaitsForColdStart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(31 * time.Second):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	client := configuredCloudRunHTTPClient(t)
	defer client.CloseIdleConnections()
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("cold-start request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.StatusCode)
	}
}

func TestCloudRunHTTPClientHonorsCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	client := configuredCloudRunHTTPClient(t)
	defer client.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(request)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected caller deadline, got %v", err)
	}
}

func configuredCloudRunHTTPClient(t *testing.T) *http.Client {
	t.Helper()
	dir := t.TempDir()
	subject := filepath.Join(dir, "subject")
	if err := os.WriteFile(subject, []byte("test-subject"), 0600); err != nil {
		t.Fatal(err)
	}
	adc := filepath.Join(dir, "adc.json")
	contents := `{"type":"external_account","audience":"//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/test/providers/test","subject_token_type":"urn:ietf:params:oauth:token-type:jwt","token_url":"https://sts.googleapis.com/v1/token","service_account_impersonation_url":"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/runner@test.iam.gserviceaccount.com:generateAccessToken","credential_source":{"file":"` + subject + `"}}`
	if err := os.WriteFile(adc, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", adc)
	t.Setenv("GOOGLE_API_USE_CLIENT_CERTIFICATE", "false")
	ctx := context.Background()
	p := &LiteLLMProvider{}
	var schema provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &schema)
	state := tfsdk.State{Schema: schema.Schema}
	data := LiteLLMProviderModel{APIBase: types.StringValue("https://gateway.run.app"), APIKey: types.StringValue("test-key"), InsecureSkipVerify: types.BoolNull(), LiteLLMChangedBy: types.StringNull(), CloudRunAudience: types.StringValue("https://gateway.run.app")}
	if diags := state.Set(ctx, &data); diags.HasError() {
		t.Fatal(diags)
	}
	var response provider.ConfigureResponse
	p.Configure(ctx, provider.ConfigureRequest{Config: tfsdk.Config{Raw: state.Raw, Schema: schema.Schema}}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	return response.ResourceData.(*Client).HTTPClient
}
