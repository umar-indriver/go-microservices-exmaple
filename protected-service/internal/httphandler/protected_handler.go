package httphandler

import (
	"encoding/json"
	"net/http"

	"github.com/microservices/protected-service/internal/client"
	"github.com/microservices/protected-service/internal/middleware"
)

type ProtectedHandler struct {
	authClient *client.AuthClient
}

func NewProtectedHandler(authClient *client.AuthClient) *ProtectedHandler {
	return &ProtectedHandler{authClient: authClient}
}

type ProtectedResponse struct {
	Message        string                 `json:"message"`
	TokenContext   middleware.UserContext `json:"token_user_context"`
	gRPCProfile    map[string]string      `json:"grpc_fetched_profile,omitempty"`
	JWKSValidated  bool                   `json:"jwks_stateless_validation"`
}

func (h *ProtectedHandler) HandleProtectedData(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized context"}`, http.StatusUnauthorized)
		return
	}

	response := ProtectedResponse{
		Message:               "Access granted! You have hit a protected endpoint.",
		TokenContext:          userCtx,
		JWKSValidated:         true,
	}

	// Inter-service gRPC lookup call to Auth service if client is available
	if h.authClient != nil {
		profileResp, err := h.authClient.GetUserProfile(r.Context(), userCtx.UserID)
		if err == nil && profileResp != nil {
			response.gRPCProfile = map[string]string{
				"user_id": profileResp.GetUserId(),
				"email":   profileResp.GetEmail(),
				"name":    profileResp.GetName(),
				"role":    profileResp.GetRole(),
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}
