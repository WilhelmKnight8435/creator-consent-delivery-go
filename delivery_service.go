package main

import (
	"context"
	"encoding/json"
	"net/http"
)

type deliveryRequest struct {
	UserID  string `json:"user_id"`
	AssetID string `json:"asset_id"`
}

type deliveryResult struct {
	UserID  string `json:"user_id"`
	AssetID string `json:"asset_id"`
	Status  string `json:"status"`
}

func deliver(ctx context.Context, client *ConsentClient, req deliveryRequest) (deliveryResult, error) {
	granted, err := client.Check(ctx, req.UserID, "third_party")
	if err != nil {
		return deliveryResult{}, err
	}
	if !granted {
		return deliveryResult{UserID: req.UserID, AssetID: req.AssetID, Status: "consent_required"}, nil
	}
	return deliveryResult{UserID: req.UserID, AssetID: req.AssetID, Status: "queued"}, nil
}

func deliveryHandler(client *ConsentClient) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req deliveryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		result, err := deliver(r.Context(), client, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})
}
