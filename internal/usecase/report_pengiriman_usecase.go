package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

const (
	reportPengirimanDateLayout = "2006-01-02"
	defaultListLimit           = int32(20)
	maxListLimit               = int32(100)
)

var (
	ErrReportPengirimanValidation         = errors.New("invalid report pengiriman payload")
	ErrReportPengirimanNotFound           = errors.New("report pengiriman not found")
	ErrReportPengirimanServiceUnavailable = errors.New("report pengiriman service unavailable")
)

type ReportPengirimanUseCase struct {
	repo entity.Querier
}

func NewReportPengirimanUseCase(repo entity.Querier) (*ReportPengirimanUseCase, error) {
	if repo == nil {
		return nil, errors.New("report pengiriman repository is required")
	}

	return &ReportPengirimanUseCase{
		repo: repo,
	}, nil
}

func (u *ReportPengirimanUseCase) Create(
	ctx context.Context,
	req model.CreateReportPengirimanRequest,
) (*model.ReportPengirimanResponse, error) {
	dateValue, err := parseReportDate(req.Date)
	if err != nil {
		return nil, err
	}

	if req.Quantity <= 0 {
		return nil, fmt.Errorf("%w: quantity must be greater than 0", ErrReportPengirimanValidation)
	}

	if req.IDWOShellSize <= 0 {
		return nil, fmt.Errorf("%w: id_wo_shell_size must be greater than 0", ErrReportPengirimanValidation)
	}

	exists, err := u.repo.WorkOrderShellSizeExists(ctx, req.IDWOShellSize)
	if err != nil {
		return nil, fmt.Errorf("%w: check work order shell size", ErrReportPengirimanServiceUnavailable)
	}
	if !exists {
		return nil, fmt.Errorf("%w: id_wo_shell_size is not registered", ErrReportPengirimanValidation)
	}

	nextID, err := u.repo.GetNextReportPengirimanID(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: get next id", ErrReportPengirimanServiceUnavailable)
	}

	created, err := u.repo.CreateReportPengiriman(ctx, entity.CreateReportPengirimanParams{
		IDReportPengiriman: nextID,
		Date:               dateValue,
		Quantity:           req.Quantity,
		IDWOShellSize:      req.IDWOShellSize,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: create report pengiriman", ErrReportPengirimanServiceUnavailable)
	}

	return mapReportPengirimanEntity(created), nil
}

func (u *ReportPengirimanUseCase) List(
	ctx context.Context,
	filter model.ListReportPengirimanFilter,
) ([]model.ReportPengirimanResponse, error) {
	dateFrom, dateTo, err := parseListDateRange(filter.DateFrom, filter.DateTo)
	if err != nil {
		return nil, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	offset := filter.Offset
	if offset < 0 {
		return nil, fmt.Errorf("%w: offset must be 0 or greater", ErrReportPengirimanValidation)
	}

	var woShellSize any
	if filter.IDWOShellSize > 0 {
		woShellSize = filter.IDWOShellSize
	}

	items, err := u.repo.ListReportPengiriman(ctx, entity.ListReportPengirimanParams{
		DateFrom:      dateFrom,
		DateTo:        dateTo,
		IDWOShellSize: woShellSize,
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: list report pengiriman", ErrReportPengirimanServiceUnavailable)
	}

	result := make([]model.ReportPengirimanResponse, 0, len(items))
	for _, item := range items {
		mapped := mapReportPengirimanEntity(item)
		if mapped != nil {
			result = append(result, *mapped)
		}
	}

	return result, nil
}

func (u *ReportPengirimanUseCase) GetByID(
	ctx context.Context,
	idReportPengiriman int32,
) (*model.ReportPengirimanResponse, error) {
	if idReportPengiriman <= 0 {
		return nil, fmt.Errorf("%w: id must be greater than 0", ErrReportPengirimanValidation)
	}

	item, err := u.repo.GetReportPengirimanByID(ctx, idReportPengiriman)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReportPengirimanNotFound
		}
		return nil, fmt.Errorf("%w: get report pengiriman by id", ErrReportPengirimanServiceUnavailable)
	}

	return mapReportPengirimanEntity(item), nil
}

func (u *ReportPengirimanUseCase) Delete(
	ctx context.Context,
	idReportPengiriman int32,
) error {
	if idReportPengiriman <= 0 {
		return fmt.Errorf("%w: id must be greater than 0", ErrReportPengirimanValidation)
	}

	affectedRows, err := u.repo.DeleteReportPengirimanByID(ctx, idReportPengiriman)
	if err != nil {
		return fmt.Errorf("%w: delete report pengiriman by id", ErrReportPengirimanServiceUnavailable)
	}
	if affectedRows == 0 {
		return ErrReportPengirimanNotFound
	}

	return nil
}

func parseReportDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("%w: date is required", ErrReportPengirimanValidation)
	}

	parsed, err := time.Parse(reportPengirimanDateLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: date must use YYYY-MM-DD", ErrReportPengirimanValidation)
	}

	return parsed, nil
}

func parseListDateRange(dateFromInput, dateToInput string) (any, any, error) {
	var dateFrom any
	if dateFromInput != "" {
		parsed, err := parseReportDate(dateFromInput)
		if err != nil {
			return nil, nil, err
		}
		dateFrom = parsed
	}

	var dateTo any
	if dateToInput != "" {
		parsed, err := parseReportDate(dateToInput)
		if err != nil {
			return nil, nil, err
		}
		dateTo = parsed
	}

	if from, ok := dateFrom.(time.Time); ok {
		if to, ok := dateTo.(time.Time); ok && from.After(to) {
			return nil, nil, fmt.Errorf("%w: date_from must be less than or equal to date_to", ErrReportPengirimanValidation)
		}
	}

	return dateFrom, dateTo, nil
}

func mapReportPengirimanEntity(item entity.ReportPengiriman) *model.ReportPengirimanResponse {
	response := &model.ReportPengirimanResponse{
		IDReportPengiriman: item.IDReportPengiriman,
	}

	if item.Date.Valid {
		response.Date = item.Date.Time.Format(reportPengirimanDateLayout)
	}

	if item.Quantity.Valid {
		response.Quantity = item.Quantity.Int32
	}

	if item.IDWOShellSize.Valid {
		response.IDWOShellSize = item.IDWOShellSize.Int32
	}

	if item.CreatedAt.Valid {
		response.CreatedAt = item.CreatedAt.Time.Format(time.RFC3339)
	}

	return response
}
