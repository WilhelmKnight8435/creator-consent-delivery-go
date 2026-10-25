package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeliverRequiresConsentBeforeQueueing(t *testing.T) {
	tests := []struct {
		name    string
		granted bool
		want    string
	}{
		{name: "granted subscriber is queued", granted: true, want: "queued"},
		{name: "ungranted subscriber is held", granted: false, want: "consent_required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v1/auth/consent/check/u-17/third_party" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				json.NewEncoder(w).Encode(map[string]any{"ok": true, "data": map[string]bool{"granted": tt.granted}, "error": nil, "metadata": map[string]any{}})
			}))
			defer server.Close()
			client := NewConsentClient(server.URL, "test-key")
			got, err := deliver(context.Background(), client, deliveryRequest{UserID: "u-17", AssetID: "asset-4"})
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != tt.want {
				t.Fatalf("status = %q, want %q", got.Status, tt.want)
			}
		})
	}
}
