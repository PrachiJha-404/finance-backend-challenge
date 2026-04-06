package user

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

func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware, adminOnlyMiddleware func(http.Handler) http.Handler) {
	r.Post("/auth/login", h.Login)
	r.Post("/users", h.CreateUser) // Allow creating users without auth for initial setup

	r.Route("/users", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/", h.ListUsers) // Admin only
		r.Get("/{id}", h.GetUser)
		r.Put("/{id}", h.UpdateUser)    // Admin only
		r.Delete("/{id}", h.DeleteUser) // Admin only
	})
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, apierr.BadRequest("invalid JSON"))
		return
	}

	user, err := h.service.CreateUser(&req)
	if err != nil {
		apierr.Write(w, apierr.BadRequest(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, apierr.BadRequest("invalid JSON"))
		return
	}

	resp, err := h.service.Login(&req)
	if err != nil {
		apierr.Write(w, apierr.Unauthorized(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		apierr.Write(w, apierr.BadRequest("invalid user ID"))
		return
	}

	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		apierr.Write(w, apierr.Unauthorized(""))
		return
	}

	// Users can only view their own profile unless admin
	if claims.Role != "admin" && claims.UserID != id {
		apierr.Write(w, apierr.Forbidden(""))
		return
	}

	user, err := h.service.GetUserByID(id)
	if err != nil {
		apierr.Write(w, apierr.NotFound("user not found"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListUsers()
	if err != nil {
		apierr.Write(w, apierr.Internal(""))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		apierr.Write(w, apierr.BadRequest("invalid user ID"))
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, apierr.BadRequest("invalid JSON"))
		return
	}

	user, err := h.service.UpdateUser(id, &req)
	if err != nil {
		apierr.Write(w, apierr.BadRequest(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		apierr.Write(w, apierr.BadRequest("invalid user ID"))
		return
	}

	err = h.service.DeleteUser(id)
	if err != nil {
		apierr.Write(w, apierr.Internal(""))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
