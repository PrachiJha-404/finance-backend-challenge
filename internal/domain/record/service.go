package record

import (
	"fmt"
	"time"

	"finance-backend-challenge/pkg/validator"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateRecord(userID int, req *CreateRecordRequest) (*Record, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}
	v := validator.New()
	v.Required("type", string(req.Type))
	v.Required("category", req.Category)
	v.Required("date", req.Date)
	if v.HasErrors() {
		return nil, fmt.Errorf("validation error: %s", v.Error())
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, use YYYY-MM-DD")
	}

	record := &Record{
		UserID:   userID,
		Amount:   req.Amount,
		Type:     req.Type,
		Category: req.Category,
		Date:     date,
		Notes:    req.Notes,
	}

	err = s.repo.Create(record)
	if err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}
	return record, nil
}

func (s *Service) GetRecordByID(id int) (*Record, error) {
	record, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("record not found")
	}
	return record, nil
}

func (s *Service) UpdateRecord(id int, req *UpdateRecordRequest) (*Record, error) {
	updates := make(map[string]interface{})

	if req.Amount != nil {
		if *req.Amount <= 0 {
			return nil, fmt.Errorf("amount must be greater than 0")
		}
		updates["amount"] = *req.Amount
	}

	if req.Type != nil {
		updates["type"] = *req.Type
	}

	if req.Category != nil {
		updates["category"] = *req.Category
	}

	if req.Date != nil {
		date, err := time.Parse("2006-01-02", *req.Date)
		if err != nil {
			return nil, fmt.Errorf("invalid date format, use YYYY-MM-DD")
		}
		updates["date"] = date
	}

	if req.Notes != nil {
		updates["notes"] = *req.Notes
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	err := s.repo.Update(id, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update record: %w", err)
	}

	return s.repo.GetByID(id)
}

func (s *Service) DeleteRecord(id int) error {
	return s.repo.Delete(id)
}

func (s *Service) ListRecords(userID int, filter *RecordFilter) ([]Record, error) {
	return s.repo.List(userID, filter)
}
