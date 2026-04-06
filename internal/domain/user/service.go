package user

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"finance-backend/internal/apierr"
	"finance-backend/pkg/validator"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Service contains all business logic for the user domain.
// It depends only on the Repository interface — never on a concrete DB type.
// This is Dependency Inversion in practice.
//
// GRASP Controller: the service acts as the system-level controller for
// user operations. HTTP handlers delegate to it; they own no logic themselves.
type Service struct {
	repo      Repository
	jwtSecret string
	jwtExpiry int // hours
}

// NewService is the factory function for Service.
// All dependencies are injected — the service never constructs its own.
func NewService(repo Repository, jwtSecret string, jwtExpiry int) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		jwtExpiry: jwtExpiry,
	}
}

// Register creates a new user account with a hashed password.
func (s *Service) Register(req *RegisterRequest) (*User, *apierr.APIError) {
	v := validator.New()
	v.Required("name", req.Name)
	v.Required("email", req.Email)
	v.IsEmail("email", req.Email)
	v.MinLength("password", req.Password, 8)
	if v.HasErrors() {
		return nil, apierr.BadRequest(v.Error())
	}

	// Check for duplicate email
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
		Role:     RoleViewer, // New users start as viewers — admin promotes them
		Status:   StatusActive,
	}

	created, err := s.repo.Create(u)
	if err != nil {
		return nil, apierr.Internal(fmt.Sprintf("failed to create user: %v", err))
	}
	return created, nil
}

// Login verifies credentials and returns a signed JWT on success.
func (s *Service) Login(req *LoginRequest) (*LoginResponse, *apierr.APIError) {
	v := validator.New()
	v.Required("email", req.Email)
	v.Required("password", req.Password)
	if v.HasErrors() {
		return nil, apierr.BadRequest(v.Error())
	}

	u, err := s.repo.GetByEmail(req.Email)
	if err != nil {
		// Deliberately vague — do not reveal whether the email exists
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

// GetByID returns a single user. Admins may fetch any user.
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

// List returns all users. Admin only — enforced at the middleware layer,
// but the service is the right place to describe *what* this operation does.
func (s *Service) List() ([]*User, *apierr.APIError) {
	users, err := s.repo.List()
	if err != nil {
		return nil, apierr.Internal("")
	}
	return users, nil
}

// Update modifies a user's name, role, or status.
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

// UpdateStatus changes only the active/inactive status of a user.
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

// --- JWT helpers ---

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

// ParseToken validates a JWT string and returns the embedded claims.
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
