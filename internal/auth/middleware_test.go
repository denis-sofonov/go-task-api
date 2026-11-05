package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRequireAuth(t *testing.T) {
	handler := NewHandler(nil, nil, []byte("test signing key"), time.Hour)
	userID := uuid.New()
	token, err := handler.issueToken(userID)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		authorization string
		wantStatus    int
	}{
		{name: "missing token", wantStatus: http.StatusUnauthorized},
		{name: "invalid token", authorization: "Bearer invalid", wantStatus: http.StatusUnauthorized},
		{name: "valid token", authorization: "Bearer " + token, wantStatus: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			mux := http.NewServeMux()
			mux.Handle("GET /tasks", handler.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if got, ok := UserIDFromContext(r.Context()); !ok || got != userID {
					t.Errorf("context user ID = %v, found = %t; want %v", got, ok, userID)
				}
				w.WriteHeader(http.StatusNoContent)
			})))

			req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d; want %d", rec.Code, tt.wantStatus)
			}
			if wantCalled := tt.wantStatus == http.StatusNoContent; called != wantCalled {
				t.Errorf("protected handler called = %t; want %t", called, wantCalled)
			}
		})
	}
}
