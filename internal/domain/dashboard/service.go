package dashboard

import (
	"fmt"

	"finance-backend-challenge/internal/apierr"

	"github.com/jmoiron/sqlx"
)

type Summary struct {
	TotalIncome   float64 `json:"total_income"   db:"total_income"`
	TotalExpenses float64 `json:"total_expenses" db:"total_expenses"`
	NetBalance    float64 `json:"net_balance"`
	RecordCount   int     `json:"record_count"   db:"record_count"`
}

// CategoryTotal holds aggregated amounts per category.
type CategoryTotal struct {
	Category string  `json:"category" db:"category"`
	Type     string  `json:"type"     db:"type"`
	Total    float64 `json:"total"    db:"total"`
	Count    int     `json:"count"    db:"count"`
}

// MonthlyTrend holds income/expense totals per calendar month.
type MonthlyTrend struct {
	Month         string  `json:"month"          db:"month"`
	TotalIncome   float64 `json:"total_income"   db:"total_income"`
	TotalExpenses float64 `json:"total_expenses" db:"total_expenses"`
	NetBalance    float64 `json:"net_balance"`
}

// RecentRecord is a lightweight projection for the recent activity feed.
type RecentRecord struct {
	ID       string  `json:"id"       db:"id"`
	Amount   float64 `json:"amount"   db:"amount"`
	Type     string  `json:"type"     db:"type"`
	Category string  `json:"category" db:"category"`
	Date     string  `json:"date"     db:"date"`
	Notes    string  `json:"notes"    db:"notes"`
}

// --- Service ---

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) GetSummary() (*Summary, *apierr.APIError) {
	var summary Summary
	err := s.db.QueryRowx(`
		SELECT
			COALESCE(SUM(CASE WHEN type = 'income'  THEN amount ELSE 0 END), 0) AS total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS total_expenses,
			COUNT(*) AS record_count
		FROM financial_records
		WHERE deleted_at IS NULL
	`).Scan(&summary.TotalIncome, &summary.TotalExpenses, &summary.RecordCount)
	if err != nil {
		return nil, apierr.Internal(fmt.Sprintf("dashboard.GetSummary: %v", err))
	}

	summary.NetBalance = summary.TotalIncome - summary.TotalExpenses
	return &summary, nil
}

func (s *Service) GetByCategory() ([]*CategoryTotal, *apierr.APIError) {
	var totals []*CategoryTotal
	err := s.db.Select(&totals, `
		SELECT
			category,
			type,
			SUM(amount) AS total,
			COUNT(*)    AS count
		FROM financial_records
		WHERE deleted_at IS NULL
		GROUP BY category, type
		ORDER BY total DESC
	`)
	if err != nil {
		return nil, apierr.Internal(fmt.Sprintf("dashboard.GetByCategory: %v", err))
	}
	return totals, nil
}

func (s *Service) GetMonthlyTrends(months int) ([]*MonthlyTrend, *apierr.APIError) {
	if months < 1 || months > 24 {
		months = 12 // sensible default
	}

	var rows []*struct {
		Month         string  `db:"month"`
		TotalIncome   float64 `db:"total_income"`
		TotalExpenses float64 `db:"total_expenses"`
	}

	err := s.db.Select(&rows, `
		SELECT
			TO_CHAR(date, 'YYYY-MM') AS month,
			COALESCE(SUM(CASE WHEN type = 'income'  THEN amount ELSE 0 END), 0) AS total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS total_expenses
		FROM financial_records
		WHERE deleted_at IS NULL
		  AND date >= DATE_TRUNC('month', NOW()) - ($1 - 1) * INTERVAL '1 month'
		GROUP BY TO_CHAR(date, 'YYYY-MM')
		ORDER BY month ASC
	`, months)
	if err != nil {
		return nil, apierr.Internal(fmt.Sprintf("dashboard.GetMonthlyTrends: %v", err))
	}

	trends := make([]*MonthlyTrend, len(rows))
	for i, row := range rows {
		trends[i] = &MonthlyTrend{
			Month:         row.Month,
			TotalIncome:   row.TotalIncome,
			TotalExpenses: row.TotalExpenses,
			NetBalance:    row.TotalIncome - row.TotalExpenses,
		}
	}
	return trends, nil
}

func (s *Service) GetRecentActivity(limit int) ([]*RecentRecord, *apierr.APIError) {
	if limit < 1 || limit > 50 {
		limit = 10
	}

	var records []*RecentRecord
	err := s.db.Select(&records, `
		SELECT id, amount, type, category, date::TEXT, COALESCE(notes, '') AS notes
		FROM financial_records
		WHERE deleted_at IS NULL
		ORDER BY date DESC, created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, apierr.Internal(fmt.Sprintf("dashboard.GetRecentActivity: %v", err))
	}
	return records, nil
}
