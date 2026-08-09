package usecase

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

const (
	qtyWoScopeWholeWO   = "WHOLE_WO"
	qtyWoScopeSize      = "SIZE"
	qtyWoScopeColor     = "COLOR"
	qtyWoScopeColorSize = "COLOR_SIZE"
)

type materialListItemApplicability struct {
	Category     *string
	QtyWoScope   *string
	IDQtyWoShell *int32
	IDQtyWoSize  *int32
}

type materialListApplicabilityRepository interface {
	MaterialListHasApplicabilityShell(context.Context, entity.MaterialListHasApplicabilityShellParams) (bool, error)
	MaterialListHasApplicabilitySize(context.Context, entity.MaterialListHasApplicabilitySizeParams) (bool, error)
	MaterialListHasApplicabilityShellSize(context.Context, entity.MaterialListHasApplicabilityShellSizeParams) (bool, error)
}

type materialListConsumptionPrefillRepository interface {
	GetMaterialListItemConsumptionPrefill(context.Context, entity.GetMaterialListItemConsumptionPrefillParams) (pgtype.Numeric, error)
}

type materialListSourceRepository interface {
	MaterialListHasSourceShell(context.Context, entity.MaterialListHasSourceShellParams) (bool, error)
	MaterialListHasSourceTrim(context.Context, entity.MaterialListHasSourceTrimParams) (bool, error)
}

func validateMaterialListItemSources(ctx context.Context, repo materialListSourceRepository, idMaterialList int32, idWoShell, idWoTrim pgtype.Int4) error {
	if idWoShell.Valid {
		exists, err := repo.MaterialListHasSourceShell(ctx, entity.MaterialListHasSourceShellParams{
			IDMaterialList: idMaterialList,
			IDWoShell:      idWoShell.Int32,
		})
		if err != nil {
			return fmt.Errorf("%w: validate source shell: %v", ErrMaterialListUnavailable, err)
		}
		if !exists {
			return ErrMaterialListValidation
		}
	}
	if idWoTrim.Valid {
		exists, err := repo.MaterialListHasSourceTrim(ctx, entity.MaterialListHasSourceTrimParams{
			IDMaterialList: idMaterialList,
			IDWoTrim:       idWoTrim.Int32,
		})
		if err != nil {
			return fmt.Errorf("%w: validate source trim: %v", ErrMaterialListUnavailable, err)
		}
		if !exists {
			return ErrMaterialListValidation
		}
	}
	return nil
}

func validateMaterialListItemApplicability(ctx context.Context, repo materialListApplicabilityRepository, idMaterialList int32, fields materialListItemApplicability) error {
	if fields.Category != nil && *fields.Category != "FABRIC" && *fields.Category != "SEWING" && *fields.Category != "PACKING" {
		return ErrMaterialListValidation
	}

	if fields.QtyWoScope == nil {
		if fields.IDQtyWoShell != nil || fields.IDQtyWoSize != nil {
			return ErrMaterialListValidation
		}
		return nil
	}

	switch *fields.QtyWoScope {
	case qtyWoScopeWholeWO:
		if fields.IDQtyWoShell != nil || fields.IDQtyWoSize != nil {
			return ErrMaterialListValidation
		}
		return nil
	case qtyWoScopeSize:
		if fields.IDQtyWoShell != nil || fields.IDQtyWoSize == nil {
			return ErrMaterialListValidation
		}
		exists, err := repo.MaterialListHasApplicabilitySize(ctx, entity.MaterialListHasApplicabilitySizeParams{
			IDMaterialList: idMaterialList,
			IDQtyWoSize:    *fields.IDQtyWoSize,
		})
		if err != nil {
			return fmt.Errorf("%w: validate applicability size: %v", ErrMaterialListUnavailable, err)
		}
		if !exists {
			return ErrMaterialListValidation
		}
		return nil
	case qtyWoScopeColor:
		if fields.IDQtyWoShell == nil || fields.IDQtyWoSize != nil {
			return ErrMaterialListValidation
		}
		exists, err := repo.MaterialListHasApplicabilityShell(ctx, entity.MaterialListHasApplicabilityShellParams{
			IDMaterialList: idMaterialList,
			IDQtyWoShell:   *fields.IDQtyWoShell,
		})
		if err != nil {
			return fmt.Errorf("%w: validate applicability shell: %v", ErrMaterialListUnavailable, err)
		}
		if !exists {
			return ErrMaterialListValidation
		}
		return nil
	case qtyWoScopeColorSize:
		if fields.IDQtyWoShell == nil || fields.IDQtyWoSize == nil {
			return ErrMaterialListValidation
		}
		exists, err := repo.MaterialListHasApplicabilityShellSize(ctx, entity.MaterialListHasApplicabilityShellSizeParams{
			IDMaterialList: idMaterialList,
			IDQtyWoShell:   *fields.IDQtyWoShell,
			IDQtyWoSize:    *fields.IDQtyWoSize,
		})
		if err != nil {
			return fmt.Errorf("%w: validate applicability shell size: %v", ErrMaterialListUnavailable, err)
		}
		if !exists {
			return ErrMaterialListValidation
		}
		return nil
	default:
		return ErrMaterialListValidation
	}
}

func materialListConsPerPCForCreate(ctx context.Context, repo materialListConsumptionPrefillRepository, explicit *float64, idWoShell, idWoTrim pgtype.Int4) (pgtype.Numeric, error) {
	if explicit != nil {
		if !validNonNegativeNumber(*explicit) {
			return pgtype.Numeric{}, ErrMaterialListValidation
		}
		return materialListNumeric(*explicit), nil
	}
	if !idWoShell.Valid && !idWoTrim.Valid {
		return pgtype.Numeric{}, nil
	}
	cons, err := repo.GetMaterialListItemConsumptionPrefill(ctx, entity.GetMaterialListItemConsumptionPrefillParams{
		IDWoTrim:  idWoTrim,
		IDWoShell: idWoShell,
	})
	if err != nil {
		return pgtype.Numeric{}, fmt.Errorf("%w: prefill consumption per pc: %v", ErrMaterialListUnavailable, err)
	}
	return cons, nil
}

func validNonNegativeNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

func materialListNumeric(value float64) pgtype.Numeric {
	var numeric pgtype.Numeric
	if err := numeric.Scan(strconv.FormatFloat(value, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}
	}
	return numeric
}

func materialListNumericPtr(value *float64) pgtype.Numeric {
	if value == nil {
		return pgtype.Numeric{}
	}
	return materialListNumeric(*value)
}

func optionalStringValue(existing pgtype.Text, update model.OptionalString) *string {
	if update.Set {
		return update.Value
	}
	if !existing.Valid {
		return nil
	}
	value := existing.String
	return &value
}

func optionalInt32Value(existing pgtype.Int4, update model.OptionalInt32) *int32 {
	if update.Set {
		return update.Value
	}
	if !existing.Valid {
		return nil
	}
	value := existing.Int32
	return &value
}

func validateUpdatedConsPerPC(update model.OptionalFloat64) error {
	if update.Set && update.Value != nil && !validNonNegativeNumber(*update.Value) {
		return ErrMaterialListValidation
	}
	return nil
}

func materialListItemResponse(id int32, item, description string, qty int32, unit string, estPrice pgtype.Numeric, idWoShell, idWoTrim pgtype.Int4, category pgtype.Text, consPerPC pgtype.Numeric, qtyWoScope pgtype.Text, idQtyWoShell, idQtyWoSize pgtype.Int4, createdAt pgtype.Timestamptz, qtySuratJalan, qtyReceived int32) model.MaterialListItemResponse {
	response := model.MaterialListItemResponse{
		ID:            id,
		Item:          item,
		Description:   description,
		Qty:           qty,
		Unit:          unit,
		EstPrice:      numericToFloat64(estPrice),
		Category:      nullableTextPtr(category),
		ConsPerPC:     numericToFloat64Ptr(consPerPC),
		QtyWoScope:    nullableTextPtr(qtyWoScope),
		IDQtyWoShell:  nullableInt32PgTypePtr(idQtyWoShell),
		IDQtyWoSize:   nullableInt32PgTypePtr(idQtyWoSize),
		CreatedAt:     createdAt.Time.Format(time.RFC3339),
		QtySuratJalan: qtySuratJalan,
		QtyReceived:   qtyReceived,
	}
	if idWoShell.Valid {
		value := idWoShell.Int32
		response.IDWoShell = &value
	}
	if idWoTrim.Valid {
		value := idWoTrim.Int32
		response.IDWoTrim = &value
	}
	return response
}

func nullableTextPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	text := value.String
	return &text
}

func nullableInt32PgTypePtr(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	integer := value.Int32
	return &integer
}
