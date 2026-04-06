package dashboard

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

func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware, analystAbove func(http.Handler) http.Handler) {
	r.Route("/dashboard", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(analystAbove)
		r.Get("/summary", h.GetSummary)
		r.Get("/categories", h.GetByCategory)
		r.Get("/trends", h.GetMonthlyTrends)
		r.Get("/recent", h.GetRecentActivity)
	})
}

func (h *Handler) GetSummary(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		apierr.Write(w, apierr.Unauthorized(""))
		return
	}

	summary, err := h.service.GetSummary(claims.UserID)
	if err != nil {
		apierr.Write(w, apierr.Internal(""))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func (h *Handler) GetByCategory(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		apierr.Write(w, apierr.Unauthorized(""))
		return
	}

	totals, err := h.service.GetByCategory(claims.UserID)
	if err != nil {
		apierr.Write(w, apierr.Internal(""))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(totals)
}

func (h *Handler) GetMonthlyTrends(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		apierr.Write(w, apierr.Unauthorized(""))
		return
	}

	monthsStr := r.URL.Query().Get("months")
	months := 12
	if m, err := strconv.Atoi(monthsStr); err == nil && m > 0 {
		months = m
	}

	trends, err := h.service.GetMonthlyTrends(claims.UserID, months)
	if err != nil {
		apierr.Write(w, apierr.Internal(""))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trends)
}

func (h *Handler) GetRecentActivity(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		apierr.Write(w, apierr.Unauthorized(""))
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	records, err := h.service.GetRecentActivity(claims.UserID, limit)
	if err != nil {
		apierr.Write(w, apierr.Internal(""))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}
