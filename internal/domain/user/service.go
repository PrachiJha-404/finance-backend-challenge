package user

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"finance-backend-challenge/pkg/validator"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo      Repository
	jwtSecret string
	jwtExpiry int // hours
}

func NewService(repo Repository, jwtSecret string, jwtExpiry int) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		jwtExpiry: jwtExpiry,
	}
}

func (s *Service) CreateUser(req *CreateUserRequest) (*User, error) {
	v := validator.New()
	v.Required("email", req.Email)
	v.IsEmail("email", req.Email)
	v.Required("password", req.Password)
	v.MinLength("password", req.Password, 6)
	v.Required("role", string(req.Role))
	if v.HasErrors() {
		return nil, fmt.Errorf("validation error: %s", v.Error())
	}

	// Check for duplicate email
	existing, err := s.repo.GetByEmail(req.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("user with this email already exists")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	u := &User{
		Email:        req.Email,
		PasswordHash: string(hashed),
		Role:         req.Role,
		Status:       StatusActive,
	}

	err = s.repo.Create(u)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return u, nil
}

func (s *Service) Login(req *LoginRequest) (*AuthResponse, error) {
	v := validator.New()
	v.Required("email", req.Email)
	v.Required("password", req.Password)
	if v.HasErrors() {
		return nil, fmt.Errorf("validation error: %s", v.Error())
	}

	u, err := s.repo.GetByEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}
	if u == nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	if u.Status != StatusActive {
		return nil, fmt.Errorf("account is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	token, err := s.generateToken(u)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResponse{Token: token, User: *u}, nil
}

func (s *Service) GetUserByID(id int) (*User, error) {
	u, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, fmt.Errorf("user not found")
	}
	return u, nil
}

func (s *Service) UpdateUser(id int, req *UpdateUserRequest) (*User, error) {
	updates := make(map[string]interface{})

	if req.Email != nil {
		v := validator.New()
		v.IsEmail("email", *req.Email)
		if v.HasErrors() {
			return nil, fmt.Errorf("validation error: %s", v.Error())
		}
		updates["email"] = *req.Email
	}

	if req.Role != nil {
		updates["role"] = *req.Role
	}

	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	err := s.repo.Update(id, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return s.repo.GetByID(id)
}

func (s *Service) DeleteUser(id int) error {
	return s.repo.Delete(id)
}

func (s *Service) ListUsers() ([]User, error) {
	return s.repo.List()
}

func (s *Service) generateToken(u *User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": u.ID,
		"email":   u.Email,
		"role":    string(u.Role),
		"exp":     time.Now().Add(time.Hour * time.Duration(s.jwtExpiry)).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *Service) ValidateToken(tokenString string) (*struct {
	ID    int
	Email string
	Role  string
}, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid user_id in token")
		}
		userID := int(userIDFloat)

		u, err := s.repo.GetByID(userID)
		if err != nil {
			return nil, err
		}
		if u == nil {
			return nil, fmt.Errorf("user not found")
		}
		return &struct {
			ID    int
			Email string
			Role  string
		}{ID: u.ID, Email: u.Email, Role: string(u.Role)}, nil
	}

	return nil, fmt.Errorf("invalid token")
}
