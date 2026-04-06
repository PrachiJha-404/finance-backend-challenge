package user

import (
	"time"
)

type Role string

const (
	RoleViewer  Role = "viewer"
	RoleAnalyst Role = "analyst"
	RoleAdmin   Role = "admin"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

type User struct {
	ID           int       `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Role         Role      `json:"role" db:"role"`
	Status       Status    `json:"status" db:"status"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Role     Role   `json:"role" validate:"required,oneof=viewer analyst admin"`
}

type UpdateUserRequest struct {
	Email  *string `json:"email,omitempty" validate:"omitempty,email"`
	Role   *Role   `json:"role,omitempty" validate:"omitempty,oneof=viewer analyst admin"`
	Status *Status `json:"status,omitempty" validate:"omitempty,oneof=active inactive"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
