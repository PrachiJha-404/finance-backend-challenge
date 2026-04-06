package dashboard

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Summary struct {
	TotalIncome   float64 `json:"total_income"`
	TotalExpenses float64 `json:"total_expenses"`
	NetBalance    float64 `json:"net_balance"`
	RecordCount   int     `json:"record_count"`
}

type CategoryTotal struct {
	Category string  `json:"category"`
	Type     string  `json:"type"`
	Total    float64 `json:"total"`
	Count    int     `json:"count"`
}

type MonthlyTrend struct {
	Month         string  `json:"month"`
	TotalIncome   float64 `json:"total_income"`
	TotalExpenses float64 `json:"total_expenses"`
	NetBalance    float64 `json:"net_balance"`
}

type RecentRecord struct {
	ID       int     `json:"id"`
	Amount   float64 `json:"amount"`
	Type     string  `json:"type"`
	Category string  `json:"category"`
	Date     string  `json:"date"`
	Notes    string  `json:"notes"`
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) GetSummary(userID int) (*Summary, error) {
	var summary Summary
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS total_expenses,
			COUNT(*) AS record_count
		FROM records
		WHERE user_id = ?`
	err := s.db.QueryRow(query, userID).Scan(&summary.TotalIncome, &summary.TotalExpenses, &summary.RecordCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}

	summary.NetBalance = summary.TotalIncome - summary.TotalExpenses
	return &summary, nil
}

func (s *Service) GetByCategory(userID int) ([]CategoryTotal, error) {
	var totals []CategoryTotal
	query := `
		SELECT
			category,
			type,
			SUM(amount) AS total,
			COUNT(*) AS count
		FROM records
		WHERE user_id = ?
		GROUP BY category, type
		ORDER BY total DESC`
	err := s.db.Select(&totals, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get category totals: %w", err)
	}
	return totals, nil
}

func (s *Service) GetMonthlyTrends(userID int, months int) ([]MonthlyTrend, error) {
	if months < 1 || months > 24 {
		months = 12
	}

	var trends []MonthlyTrend
	query := `
		SELECT
			strftime('%Y-%m', date) AS month,
			SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END) AS total_income,
			SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END) AS total_expenses
		FROM records
		WHERE user_id = ?
		GROUP BY strftime('%Y-%m', date)
		ORDER BY month DESC
		LIMIT ?`
	err := s.db.Select(&trends, query, userID, months)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly trends: %w", err)
	}

	for i := range trends {
		trends[i].NetBalance = trends[i].TotalIncome - trends[i].TotalExpenses
	}
	return trends, nil
}

func (s *Service) GetRecentActivity(userID int, limit int) ([]RecentRecord, error) {
	if limit < 1 || limit > 50 {
		limit = 10
	}

	var records []RecentRecord
	query := `
		SELECT id, amount, type, category, date, COALESCE(notes, '') AS notes
		FROM records
		WHERE user_id = ?
		ORDER BY date DESC, created_at DESC
		LIMIT ?`
	err := s.db.Select(&records, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent activity: %w", err)
	}
	return records, nil
}
