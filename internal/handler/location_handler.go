package handler

import (
	"encoding/json"
	"net/http"

	"github.com/mhmddaud/transjakarta-fleet/internal/model"
	"github.com/mhmddaud/transjakarta-fleet/internal/service"
)

type LocationHandler struct {
	locationService *service.LocationService
}

func NewLocationHandler(svc *service.LocationService) *LocationHandler {
	return &LocationHandler{locationService: svc}
}

func (h *LocationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	var location model.VehicleLocation
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&location); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	var extra any
	if err := decoder.Decode(&extra); err == nil {
		http.Error(w, "request must contain a single JSON object", http.StatusBadRequest)
		return
	}

	event, err := h.locationService.ProcessLocation(r.Context(), &location)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]any{
		"message": "location processed",
		"event":   event,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}
