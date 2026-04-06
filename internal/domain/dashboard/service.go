package user

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"finance-backend-challenge/internal/apierr"
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

func (s *Service) Register(req *RegisterRequest) (*User, *apierr.APIError) {
	v := validator.New()
	v.Required("name", req.Name)
	v.Required("email", req.Email)
	v.IsEmail("email", req.Email)
	v.MinLength("password", req.Password, 8)
	if v.HasErrors() {
		return nil, apierr.BadRequest(v.Error())
	}

	existing, err := s.repo.GetByEmail(req.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, apierr.Internal("")
	}
	if existing != nil {
		return nil, apierr.Conflict("a user with this email already exists")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apierr.Internal("failed to process password")
	}

	u := &User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashed),
		Role:     RoleViewer,
		Status:   StatusActive,
	}

	created, err := s.repo.Create(u)
	if err != nil {
		return nil, apierr.Internal(fmt.Sprintf("failed to create user: %v", err))
	}
	return created, nil
}

func (s *Service) Login(req *LoginRequest) (*LoginResponse, *apierr.APIError) {
	v := validator.New()
	v.Required("email", req.Email)
	v.Required("password", req.Password)
	if v.HasErrors() {
		return nil, apierr.BadRequest(v.Error())
	}

	u, err := s.repo.GetByEmail(req.Email)
	if err != nil {
		return nil, apierr.Unauthorized("invalid email or password")
	}

	if !u.IsActive() {
		return nil, apierr.Forbidden("this account has been deactivated")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)); err != nil {
		return nil, apierr.Unauthorized("invalid email or password")
	}

	token, err := s.generateToken(u)
	if err != nil {
		return nil, apierr.Internal("failed to generate token")
	}

	return &LoginResponse{Token: token, User: u}, nil
}

func (s *Service) GetByID(id string) (*User, *apierr.APIError) {
	u, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apierr.NotFound("user not found")
		}
		return nil, apierr.Internal("")
	}
	return u, nil
}

func (s *Service) List() ([]*User, *apierr.APIError) {
	users, err := s.repo.List()
	if err != nil {
		return nil, apierr.Internal("")
	}
	return users, nil
}

func (s *Service) Update(id string, req *UpdateRequest) (*User, *apierr.APIError) {
	v := validator.New()
	v.Required("name", req.Name)
	if req.Role != "" {
		v.OneOf("role", req.Role, string(RoleViewer), string(RoleAnalyst), string(RoleAdmin))
	}
	if req.Status != "" {
		v.OneOf("status", req.Status, string(StatusActive), string(StatusInactive))
	}
	if v.HasErrors() {
		return nil, apierr.BadRequest(v.Error())
	}

	u, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apierr.NotFound("user not found")
		}
		return nil, apierr.Internal("")
	}

	u.Name = req.Name
	if req.Role != "" {
		u.Role = Role(req.Role)
	}
	if req.Status != "" {
		u.Status = Status(req.Status)
	}

	updated, err := s.repo.Update(u)
	if err != nil {
		return nil, apierr.Internal("")
	}
	return updated, nil
}

func (s *Service) UpdateStatus(id string, req *UpdateStatusRequest) (*User, *apierr.APIError) {
	v := validator.New()
	v.OneOf("status", req.Status, string(StatusActive), string(StatusInactive))
	if v.HasErrors() {
		return nil, apierr.BadRequest(v.Error())
	}

	u, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apierr.NotFound("user not found")
		}
		return nil, apierr.Internal("")
	}

	u.Status = Status(req.Status)
	updated, err := s.repo.Update(u)
	if err != nil {
		return nil, apierr.Internal("")
	}
	return updated, nil
}

type Claims struct {
	UserID string `json:"user_id"`
	Role   Role   `json:"role"`
	jwt.RegisteredClaims
}

func (s *Service) generateToken(u *User) (string, error) {
	claims := Claims{
		UserID: u.ID,
		Role:   u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.jwtExpiry) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *Service) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
