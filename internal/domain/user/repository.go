package user

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Repository interface {
	Create(user *User) error
	GetByID(id int) (*User, error)
	GetByEmail(email string) (*User, error)
	Update(id int, updates map[string]interface{}) error
	Delete(id int) error
	List() ([]User, error)
}

type postgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(user *User) error {
	query := `
		INSERT INTO users (email, password_hash, role, status)
		VALUES (?, ?, ?, ?)`
	result, err := r.db.Exec(query, user.Email, user.PasswordHash, user.Role, user.Status)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = int(id)
	return r.db.Get(user, "SELECT created_at, updated_at FROM users WHERE id = ?", user.ID)
}

func (r *postgresRepository) GetByID(id int) (*User, error) {
	var user User
	query := `SELECT id, email, password_hash, role, status, created_at, updated_at FROM users WHERE id = ?`
	err := r.db.Get(&user, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (r *postgresRepository) GetByEmail(email string) (*User, error) {
	var user User
	query := `SELECT id, email, password_hash, role, status, created_at, updated_at FROM users WHERE email = ?`
	err := r.db.Get(&user, query, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (r *postgresRepository) Update(id int, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	query := "UPDATE users SET "
	args := []interface{}{}
	i := 1
	for field, value := range updates {
		query += fmt.Sprintf("%s = $%d, ", field, i)
		args = append(args, value)
		i++
	}
	query += "updated_at = CURRENT_TIMESTAMP WHERE id = ?"
	args = append(args, id)

	_, err := r.db.Exec(query, args...)
	return err
}

func (r *postgresRepository) Delete(id int) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *postgresRepository) List() ([]User, error) {
	var users []User
	query := `SELECT id, email, password_hash, role, status, created_at, updated_at FROM users ORDER BY created_at DESC`
	err := r.db.Select(&users, query)
	return users, err
}
