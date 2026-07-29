package auth

import "net/http"

// DisabledAuthenticator describes an intentionally unauthenticated appliance
// API. It is used only by the isolated NoAuth appliance edition so its browser
// client can render the dashboard without falling back to a Google login.
type DisabledAuthenticator struct{}

func NewDisabledAuthenticator() *DisabledAuthenticator { return &DisabledAuthenticator{} }

// Register mounts the minimal discovery endpoint. All monitoring routes remain
// mounted directly on the API mux and therefore require no session or token.
func (a *DisabledAuthenticator) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/method", func(w http.ResponseWriter, _ *http.Request) {
		writeJSONAuth(w, http.StatusOK, map[string]string{"provider": "disabled"})
	})
}
