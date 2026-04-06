package record

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type Repository interface {
	Create(record *Record) error
	GetByID(id int) (*Record, error)
	Update(id int, updates map[string]interface{}) error
	Delete(id int) error
	List(userID int, filter *RecordFilter) ([]Record, error)
}

type postgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(record *Record) error {
	query := `
		INSERT INTO records (user_id, amount, type, category, date, notes)
		VALUES (?, ?, ?, ?, ?, ?)`
	result, err := r.db.Exec(query, record.UserID, record.Amount, record.Type, record.Category, record.Date, record.Notes)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	record.ID = int(id)
	return r.db.Get(record, "SELECT created_at, updated_at FROM records WHERE id = ?", record.ID)
}

func (r *postgresRepository) GetByID(id int) (*Record, error) {
	var record Record
	query := `SELECT id, user_id, amount, type, category, date, notes, created_at, updated_at FROM records WHERE id = ?`
	err := r.db.Get(&record, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &record, err
}

func (r *postgresRepository) Update(id int, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	query := "UPDATE records SET "
	args := []interface{}{}
	i := 1
	for field, value := range updates {
		query += fmt.Sprintf("%s = ?, ", field)
		args = append(args, value)
		i++
	}
	query += "updated_at = CURRENT_TIMESTAMP WHERE id = ?"
	args = append(args, id)

	_, err := r.db.Exec(query, args...)
	return err
}

func (r *postgresRepository) Delete(id int) error {
	query := `DELETE FROM records WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *postgresRepository) List(userID int, filter *RecordFilter) ([]Record, error) {
	conditions := []string{"user_id = ?"}
	args := []interface{}{userID}
	argIdx := 2

	if filter != nil {
		if filter.Type != nil {
			conditions = append(conditions, fmt.Sprintf("type = ?", argIdx))
			args = append(args, *filter.Type)
			argIdx++
		}
		if filter.Category != nil {
			conditions = append(conditions, fmt.Sprintf("category LIKE ?", argIdx))
			args = append(args, "%"+*filter.Category+"%")
			argIdx++
		}
		if filter.StartDate != nil {
			conditions = append(conditions, fmt.Sprintf("date >= ?", argIdx))
			date, _ := time.Parse("2006-01-02", *filter.StartDate)
			args = append(args, date)
			argIdx++
		}
		if filter.EndDate != nil {
			conditions = append(conditions, fmt.Sprintf("date <= ?", argIdx))
			date, _ := time.Parse("2006-01-02", *filter.EndDate)
			args = append(args, date)
			argIdx++
		}
	}

	where := "WHERE " + strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`
		SELECT id, user_id, amount, type, category, date, notes, created_at, updated_at
		FROM records
		%s
		ORDER BY date DESC, created_at DESC`, where)

	var records []Record
	err := r.db.Select(&records, query, args...)
	return records, err
}
