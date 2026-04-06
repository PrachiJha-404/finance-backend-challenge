package record

import (
	"encoding/json"
	"net/http"
	"strconv"

	"finance-backend-challenge/internal/apierr"
	"finance-backend-challenge/internal/middleware"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware, anyRoleMiddleware, adminOnlyMiddleware func(http.Handler) http.Handler) {
	r.Route("/records", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(anyRoleMiddleware)
		r.Post("/", h.CreateRecord)
		r.Get("/", h.ListRecords)
		r.Get("/{id}", h.GetRecord)
		r.Put("/{id}", h.UpdateRecord)
		r.Delete("/{id}", h.DeleteRecord)
	})
}

func (h *Handler) CreateRecord(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		apierr.Write(w, apierr.Unauthorized(""))
		return
	}

	var req CreateRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, apierr.BadRequest("invalid JSON"))
		return
	}

	record, err := h.service.CreateRecord(claims.UserID, &req)
	if err != nil {
		apierr.Write(w, apierr.BadRequest(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(record)
}

func (h *Handler) GetRecord(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		apierr.Write(w, apierr.Unauthorized(""))
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		apierr.Write(w, apierr.BadRequest("invalid record ID"))
		return
	}

	record, err := h.service.GetRecordByID(id)
	if err != nil {
		apierr.Write(w, apierr.NotFound("record not found"))
		return
	}

	// Users can only access their own records unless admin
	if string(claims.Role) != "admin" && record.UserID != claims.UserID {
		apierr.Write(w, apierr.Forbidden(""))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

func (h *Handler) ListRecords(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		apierr.Write(w, apierr.Unauthorized(""))
		return
	}

	// Parse query parameters for filtering
	filter := &RecordFilter{}
	if typ := r.URL.Query().Get("type"); typ != "" {
		t := Type(typ)
		filter.Type = &t
	}
	if category := r.URL.Query().Get("category"); category != "" {
		filter.Category = &category
	}
	if startDate := r.URL.Query().Get("start_date"); startDate != "" {
		filter.StartDate = &startDate
	}
	if endDate := r.URL.Query().Get("end_date"); endDate != "" {
		filter.EndDate = &endDate
	}

	records, err := h.service.ListRecords(claims.UserID, filter)
	if err != nil {
		apierr.Write(w, apierr.Internal(""))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

func (h *Handler) UpdateRecord(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		apierr.Write(w, apierr.Unauthorized(""))
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		apierr.Write(w, apierr.BadRequest("invalid record ID"))
		return
	}

	// Check ownership
	record, err := h.service.GetRecordByID(id)
	if err != nil {
		apierr.Write(w, apierr.NotFound("record not found"))
		return
	}
	if string(claims.Role) != "admin" && record.UserID != claims.UserID {
		apierr.Write(w, apierr.Forbidden(""))
		return
	}

	var req UpdateRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, apierr.BadRequest("invalid JSON"))
		return
	}

	record, err = h.service.UpdateRecord(id, &req)
	if err != nil {
		apierr.Write(w, apierr.BadRequest(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

func (h *Handler) DeleteRecord(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		apierr.Write(w, apierr.Unauthorized(""))
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		apierr.Write(w, apierr.BadRequest("invalid record ID"))
		return
	}

	// Check ownership
	record, err := h.service.GetRecordByID(id)
	if err != nil {
		apierr.Write(w, apierr.NotFound("record not found"))
		return
	}
	if string(claims.Role) != "admin" && record.UserID != claims.UserID {
		apierr.Write(w, apierr.Forbidden(""))
		return
	}

	err = h.service.DeleteRecord(id)
	if err != nil {
		apierr.Write(w, apierr.Internal(""))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
