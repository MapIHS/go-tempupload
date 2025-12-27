package helpers

import (
	"encoding/json"
	"net/http"

	"github.com/MapIHS/tempuploud/types"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteDATA[T any](w http.ResponseWriter, status int, data T) {
	resp := types.APIResponse[T]{Data: &data}
	WriteJSON(w, status, resp)
}

func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, types.APIResponse[any]{Error: msg})
}

func WriteNotFound(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotFound, "Not Found")
}

func WriteMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
}
