package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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

// ServeVehicleHTTP menangani endpoint lokasi terakhir dan riwayat kendaraan.
func (h *LocationHandler) ServeVehicleHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.Trim(strings.TrimSpace(r.URL.Path), "/")
	parts := strings.Split(path, "/")

	// Format:
	// /vehicles/{vehicle_id}/location
	// /vehicles/{vehicle_id}/history
	if len(parts) != 3 || parts[0] != "vehicles" ||
		parts[1] == "" {
		http.NotFound(w, r)
		return
	}

	vehicleID := parts[1]

	w.Header().Set("Content-Type", "application/json")

	switch parts[2] {
	case "location":
		location, err := h.locationService.GetLatestLocation(
			r.Context(),
			vehicleID,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if location == nil {
			http.Error(w, "vehicle location not found", http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(location); err != nil {
			return
		}

	case "history":
		query := r.URL.Query()

		start, err := strconv.ParseInt(query.Get("start"), 10, 64)
		if err != nil {
			http.Error(w, "invalid start: expected Unix timestamp in seconds", http.StatusBadRequest)
			return
		}

		end, err := strconv.ParseInt(query.Get("end"), 10, 64)
		if err != nil {
			http.Error(w, "invalid end: expected Unix timestamp in seconds", http.StatusBadRequest)
			return
		}

		locations, err := h.locationService.GetLocationHistory(
			r.Context(),
			vehicleID,
			start,
			end,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := json.NewEncoder(w).Encode(locations); err != nil {
			return
		}

	default:
		http.Error(w, fmt.Sprintf("unknown vehicle endpoint: %s", parts[2]), http.StatusNotFound)
	}
}
