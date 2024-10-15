package main

import (
	"context"
	"testing"

	"github.com/hashicorp/go-hclog"
	credentialcomposerv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/plugin/server/credentialcomposer/v1"
	configv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/service/common/config/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// configurePlugin sets up the plugin configuration for tests
func configurePlugin(t *testing.T, plugin *Plugin) {
	req := &configv1.ConfigureRequest{
		HclConfiguration: "", // Use an empty configuration or provide necessary config if needed
	}

	_, err := plugin.Configure(context.Background(), req)
	if err != nil {
		t.Fatalf("Failed to configure plugin: %v", err)
	}
}

// TestComposeWorkloadJWTSVID tests the ComposeWorkloadJWTSVID method of the Plugin.
func TestComposeWorkloadJWTSVID(t *testing.T) {
	plugin := &Plugin{}
	plugin.SetLogger(hclog.NewNullLogger()) // Set a no-op logger for testing

	// Configure the plugin before running the tests
	configurePlugin(t, plugin)

	// Define test cases
	tests := []struct {
		name           string
		request        *credentialcomposerv1.ComposeWorkloadJWTSVIDRequest
		expectedErr    codes.Code
		expectedClaims map[string]interface{}
	}{
		{
			name: "valid SPIFFE ID",
			request: &credentialcomposerv1.ComposeWorkloadJWTSVIDRequest{
				SpiffeId: "spiffe://example.org/workload",
				Attributes: &credentialcomposerv1.JWTSVIDAttributes{
					Claims: &structpb.Struct{},
				},
			},
			expectedErr: codes.OK,
			expectedClaims: map[string]interface{}{
				"spiffe-id":    "spiffe://example.org/workload",
				"trust-domain": "example.org",
				"workload":     "workload",
			},
		},
		{
			name:        "missing SPIFFE ID",
			request:     &credentialcomposerv1.ComposeWorkloadJWTSVIDRequest{},
			expectedErr: codes.InvalidArgument,
		},
		{
			name: "invalid SPIFFE ID format",
			request: &credentialcomposerv1.ComposeWorkloadJWTSVIDRequest{
				SpiffeId: "invalid-spiffe-id",
				Attributes: &credentialcomposerv1.JWTSVIDAttributes{
					Claims: &structpb.Struct{},
				},
			},
			expectedErr: codes.InvalidArgument,
		},
	}

	// Execute test cases
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := plugin.ComposeWorkloadJWTSVID(context.Background(), test.request)

			if test.expectedErr == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, response)

				// Assert that the claims were added as expected
				actualClaims := response.Attributes.GetClaims().AsMap()
				assert.Equal(t, test.expectedClaims, actualClaims)
			} else {
				// Assert error code
				require.Error(t, err)
				st, _ := status.FromError(err)
				assert.Equal(t, test.expectedErr, st.Code())
			}
		})
	}
}

// TestUpdateClaims tests the updateClaims function.
func TestUpdateClaims(t *testing.T) {
	originalClaims, err := structpb.NewStruct(map[string]interface{}{
		"existing-claim": "original-value",
	})
	require.NoError(t, err)

	updates := map[string]interface{}{
		"new-claim":      "new-value",
		"existing-claim": "updated-value",
	}

	updatedClaims, err := updateClaims(originalClaims, updates)
	require.NoError(t, err)

	expectedClaims := map[string]interface{}{
		"existing-claim": "updated-value",
		"new-claim":      "new-value",
	}

	actualClaims := updatedClaims.AsMap()
	assert.Equal(t, expectedClaims, actualClaims)
}

// TestParseTrustDomainAndWorkload tests the parseTrustDomainAndWorkload function.
func TestParseTrustDomainAndWorkload(t *testing.T) {
	tests := []struct {
		name             string
		spiffeID         string
		expectedDomain   string
		expectedWorkload string
		expectError      bool
	}{
		{
			name:             "valid SPIFFE ID",
			spiffeID:         "spiffe://example.org/workload",
			expectedDomain:   "example.org",
			expectedWorkload: "workload",
			expectError:      false,
		},
		{
			name:        "invalid SPIFFE ID format",
			spiffeID:    "invalid-spiffe-id",
			expectError: true,
		},
		{
			name:        "missing workload path",
			spiffeID:    "spiffe://example.org",
			expectError: true,
		},
		{
			name:        "missing trust domain",
			spiffeID:    "spiffe:///workload",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			trustDomain, workload, err := parseTrustDomainAndWorkload(test.spiffeID)

			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedDomain, trustDomain)
				assert.Equal(t, test.expectedWorkload, workload)
			}
		})
	}
}
