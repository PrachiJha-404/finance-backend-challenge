package record

import (
	"time"
)

type Type string

const (
	TypeIncome  Type = "income"
	TypeExpense Type = "expense"
)

type Record struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Amount    float64   `json:"amount" db:"amount"`
	Type      Type      `json:"type" db:"type"`
	Category  string    `json:"category" db:"category"`
	Date      time.Time `json:"date" db:"date"`
	Notes     string    `json:"notes" db:"notes"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type CreateRecordRequest struct {
	Amount   float64 `json:"amount" validate:"required,gt=0"`
	Type     Type    `json:"type" validate:"required,oneof=income expense"`
	Category string  `json:"category" validate:"required"`
	Date     string  `json:"date" validate:"required"` // YYYY-MM-DD
	Notes    string  `json:"notes,omitempty"`
}

type UpdateRecordRequest struct {
	Amount   *float64 `json:"amount,omitempty" validate:"omitempty,gt=0"`
	Type     *Type    `json:"type,omitempty" validate:"omitempty,oneof=income expense"`
	Category *string  `json:"category,omitempty" validate:"omitempty"`
	Date     *string  `json:"date,omitempty" validate:"omitempty"` // YYYY-MM-DD
	Notes    *string  `json:"notes,omitempty"`
}

type RecordFilter struct {
	Type      *Type   `json:"type,omitempty"`
	Category  *string `json:"category,omitempty"`
	StartDate *string `json:"start_date,omitempty"` // YYYY-MM-DD
	EndDate   *string `json:"end_date,omitempty"`   // YYYY-MM-DD
}
