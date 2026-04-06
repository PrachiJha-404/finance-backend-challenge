package record

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type Reader interface {
	GetByID(id string) (*Record, error)
	List(f Filter) ([]*Record, int, error)
}

type Writer interface {
	Create(r *Record) (*Record, error)
	Update(r *Record) (*Record, error)
	SoftDelete(id string) error
}

type Repository interface {
	Reader
	Writer
}

type postgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) GetByID(id string) (*Record, error) {
	var rec Record
	err := r.db.Get(&rec, `
		SELECT id, created_by, amount, type, category, date, notes,
		       deleted_at, created_at, updated_at
		FROM financial_records
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return nil, fmt.Errorf("record.GetByID: %w", err)
	}
	return &rec, nil
}

func (r *postgresRepository) List(f Filter) ([]*Record, int, error) {
	conditions := []string{"deleted_at IS NULL"}
	args := []any{}
	argIdx := 1

	if f.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, f.Type)
		argIdx++
	}
	if f.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category ILIKE $%d", argIdx))
		args = append(args, "%"+f.Category+"%")
		argIdx++
	}
	if f.From != "" {
		conditions = append(conditions, fmt.Sprintf("date >= $%d", argIdx))
		args = append(args, f.From)
		argIdx++
	}
	if f.To != "" {
		conditions = append(conditions, fmt.Sprintf("date <= $%d", argIdx))
		args = append(args, f.To)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM financial_records %s", where)
	if err := r.db.Get(&total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("record.List count: %w", err)
	}

	offset := (f.Page - 1) * f.Limit
	dataQuery := fmt.Sprintf(`
		SELECT id, created_by, amount, type, category, date, notes,
		       deleted_at, created_at, updated_at
		FROM financial_records
		%s
		ORDER BY date DESC, created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, f.Limit, offset)

	var records []*Record
	if err := r.db.Select(&records, dataQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("record.List: %w", err)
	}

	return records, total, nil
}

func (r *postgresRepository) Create(rec *Record) (*Record, error) {
	var created Record
	err := r.db.QueryRowx(`
		INSERT INTO financial_records (created_by, amount, type, category, date, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_by, amount, type, category, date, notes,
		          deleted_at, created_at, updated_at
	`, rec.CreatedBy, rec.Amount, rec.Type, rec.Category, rec.Date, rec.Notes).
		StructScan(&created)
	if err != nil {
		return nil, fmt.Errorf("record.Create: %w", err)
	}
	return &created, nil
}

func (r *postgresRepository) Update(rec *Record) (*Record, error) {
	rec.UpdatedAt = time.Now()
	var updated Record
	err := r.db.QueryRowx(`
		UPDATE financial_records
		SET amount = $1, type = $2, category = $3, date = $4, notes = $5, updated_at = $6
		WHERE id = $7 AND deleted_at IS NULL
		RETURNING id, created_by, amount, type, category, date, notes,
		          deleted_at, created_at, updated_at
	`, rec.Amount, rec.Type, rec.Category, rec.Date, rec.Notes, rec.UpdatedAt, rec.ID).
		StructScan(&updated)
	if err != nil {
		return nil, fmt.Errorf("record.Update: %w", err)
	}
	return &updated, nil
}

func (r *postgresRepository) SoftDelete(id string) error {
	result, err := r.db.Exec(`
		UPDATE financial_records
		SET deleted_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`, time.Now(), id)
	if err != nil {
		return fmt.Errorf("record.SoftDelete: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}
