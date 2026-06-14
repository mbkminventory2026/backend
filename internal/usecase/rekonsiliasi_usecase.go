package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

var (
	ErrRekonsiliasiValidation      = errors.New("invalid rekonsiliasi payload")
	ErrRekonsiliasiNotFound        = errors.New("rekonsiliasi not found")
	ErrRekonsiliasiAlreadyExists   = errors.New("rekonsiliasi already exists for this work order")
	ErrRekonsiliasiSourceNotFound  = errors.New("rekonsiliasi source data not found")
	ErrRekonsiliasiUnavailable     = errors.New("rekonsiliasi service unavailable")
	rekonsiliasiSortColumns        = buildSortWhitelist("created_at", "updated_at", "id_rekonsiliasi", "id_wo", "buyer", "style")
)

type RekonsiliasiUseCase struct {
	repo   entity.Querier
	dbPool *pgxpool.Pool
}

type rekonsiliasiSnapshot struct {
	header         entity.GetRekonsiliasiSourceHeaderRow
	consSummary    string
	namaBahan      string
	warnaKain      []string
	colorSummaries []model.RekonsiliasiColorSummaryResponse
	sourceRows     []rekonsiliasiMaterialSeed
}

type rekonsiliasiMaterialSeed struct {
	RowNo                int32
	Kategori             string
	Description          string
	SizeLabel            string
	RatioSource          float64
	QtyWO                int32
	Toleransi            int32
	Satuan               string
	QtyActualKirimSource int32
	IDMaterialListItem   *int32
	IDWoShell            *int32
	IDWoTrim             *int32
}

type rekonsiliasiManualState struct {
	RatioInput           float64
	QtyPerPcsInput       float64
	QtyActualKirimManual int32
	RejectQty            int32
	ReturQty             int32
	Keterangan           string
	TerimaEntries        []model.UpdateRekonsiliasiTerimaEntryRequest
}

func NewRekonsiliasiUseCase(repo entity.Querier, dbPool *pgxpool.Pool) (*RekonsiliasiUseCase, error) {
	if repo == nil {
		return nil, errors.New("rekonsiliasi repository is required")
	}
	if dbPool == nil {
		return nil, errors.New("database pool is required")
	}

	return &RekonsiliasiUseCase{
		repo:   repo,
		dbPool: dbPool,
	}, nil
}

func (u *RekonsiliasiUseCase) ListRekonsiliasis(ctx context.Context, filter model.RekonsiliasiListFilter) (*model.RekonsiliasiListResponse, error) {
	page, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter.ListQueryFilter, "updated_at", true, rekonsiliasiSortColumns)

	rows, err := u.repo.ListRekonsiliasis(ctx, entity.ListRekonsiliasisParams{
		SearchTerm: search,
		IDWo:       nullableInt32Param(filter.IDWo),
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageOffset: offset,
		PageLimit:  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list rekonsiliasi", ErrRekonsiliasiUnavailable)
	}

	items := make([]model.RekonsiliasiListItem, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, model.RekonsiliasiListItem{
			IDRekonsiliasi:    row.IDRekonsiliasi,
			IDWo:              row.IDWo,
			NamaWo:            row.NamaWo,
			Buyer:             row.Buyer,
			Brand:             row.Brand,
			Style:             row.Style,
			QtyPo:             row.QtyPo,
			PlanCutTotal:      row.PlanCutTotal,
			CreatedAt:         row.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:         row.UpdatedAt.Time.Format(time.RFC3339),
			CreatedByUsername: row.CreatedByUsername,
			UpdatedByUsername: row.UpdatedByUsername,
		})
	}

	return &model.RekonsiliasiListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func (u *RekonsiliasiUseCase) CreateRekonsiliasi(ctx context.Context, userID int32, req model.CreateRekonsiliasiRequest) (*model.RekonsiliasiDetailResponse, error) {
	if req.IDWo <= 0 {
		return nil, ErrRekonsiliasiValidation
	}

	if _, err := u.repo.GetRekonsiliasiByWOID(ctx, req.IDWo); err == nil {
		return nil, ErrRekonsiliasiAlreadyExists
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: failed to check existing rekonsiliasi", ErrRekonsiliasiUnavailable)
	}

	snapshot, err := u.buildSnapshot(ctx, req.IDWo)
	if err != nil {
		return nil, err
	}

	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to start transaction", ErrRekonsiliasiUnavailable)
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()

	qtx := entity.New(tx)
	header, err := qtx.CreateRekonsiliasi(ctx, entity.CreateRekonsiliasiParams{
		IDWo:             snapshot.header.IDWo,
		Jasa:             snapshot.header.Jasa,
		NoPo:             snapshot.header.NoPo,
		Delivery:         snapshot.header.Delivery,
		Buyer:            snapshot.header.Buyer,
		Brand:            snapshot.header.Brand,
		Style:            snapshot.header.Style,
		QtyPo:            snapshot.header.QtyPo,
		PlanCutTotal:     snapshot.header.PlanCutTotal,
		ConsBajuSummary:  snapshot.consSummary,
		NamaBahan:        snapshot.namaBahan,
		WarnaKainSummary: snapshot.warnaKain,
		CreatedBy:        pgtype.Int4{Int32: userID, Valid: true},
		UpdatedBy:        pgtype.Int4{Int32: userID, Valid: true},
	})
	if err != nil {
		return nil, mapRekonsiliasiDBError(err)
	}

	for _, sourceRow := range snapshot.sourceRows {
		if _, err := qtx.CreateRekonsiliasiMaterialRow(ctx, entity.CreateRekonsiliasiMaterialRowParams{
			IDRekonsiliasi:        header.IDRekonsiliasi,
			RowNo:                 sourceRow.RowNo,
			Kategori:              sourceRow.Kategori,
			Description:           sourceRow.Description,
			SizeLabel:             sourceRow.SizeLabel,
			RatioSource:           mustNumeric(sourceRow.RatioSource),
			RatioInput:            mustNumeric(sourceRow.RatioSource),
			QtyPerPcsInput:        mustNumeric(0),
			QtyWo:                 sourceRow.QtyWO,
			Toleransi:             sourceRow.Toleransi,
			Satuan:                sourceRow.Satuan,
			QtyActualKirimSource:  sourceRow.QtyActualKirimSource,
			QtyActualKirimManual:  0,
			RejectQty:             0,
			ReturQty:              0,
			Keterangan:            "",
			IDMaterialListItem:    nullableInt32Param(sourceRow.IDMaterialListItem),
			IDWoShell:             nullableInt32Param(sourceRow.IDWoShell),
			IDWoTrim:              nullableInt32Param(sourceRow.IDWoTrim),
		}); err != nil {
			return nil, fmt.Errorf("%w: failed to create rekonsiliasi material row", ErrRekonsiliasiUnavailable)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit create rekonsiliasi", ErrRekonsiliasiUnavailable)
	}

	return u.GetRekonsiliasi(ctx, header.IDRekonsiliasi)
}

func (u *RekonsiliasiUseCase) GetRekonsiliasi(ctx context.Context, idRekonsiliasi int32) (*model.RekonsiliasiDetailResponse, error) {
	header, err := u.repo.GetRekonsiliasiByID(ctx, idRekonsiliasi)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRekonsiliasiNotFound
		}
		return nil, fmt.Errorf("%w: failed to get rekonsiliasi header", ErrRekonsiliasiUnavailable)
	}

	colorRows, err := u.repo.ListRekonsiliasiColorSourcesByWO(ctx, header.IDWo)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get rekonsiliasi colors", ErrRekonsiliasiUnavailable)
	}

	materialRows, err := u.repo.ListRekonsiliasiMaterialRowsByRekonsiliasiID(ctx, idRekonsiliasi)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get rekonsiliasi material rows", ErrRekonsiliasiUnavailable)
	}

	terimaEntries, err := u.repo.ListRekonsiliasiTerimaEntriesByRekonsiliasiID(ctx, idRekonsiliasi)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get rekonsiliasi terima entries", ErrRekonsiliasiUnavailable)
	}

	warnaSummary, err := decodeWarnaSummary(header.WarnaKainSummary)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode warna summary", ErrRekonsiliasiUnavailable)
	}

	return &model.RekonsiliasiDetailResponse{
		Header: model.RekonsiliasiHeaderResponse{
			IDRekonsiliasi:    header.IDRekonsiliasi,
			IDWo:              header.IDWo,
			Jasa:              header.Jasa,
			NoPO:              header.NoPo,
			Delivery:          formatDate(header.Delivery),
			Buyer:             header.Buyer,
			Brand:             header.Brand,
			Style:             header.Style,
			QtyPO:             header.QtyPo,
			PlanCutTotal:      header.PlanCutTotal,
			ConsBajuSummary:   header.ConsBajuSummary,
			NamaBahan:         header.NamaBahan,
			WarnaKainSummary:  warnaSummary,
			CreatedAt:         header.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:         header.UpdatedAt.Time.Format(time.RFC3339),
			CreatedByUsername: header.CreatedByUsername,
			UpdatedByUsername: header.UpdatedByUsername,
		},
		ColorSummaries: buildRekonsiliasiColorSummaries(colorRows),
		MaterialRows:   buildRekonsiliasiMaterialRows(materialRows, terimaEntries),
	}, nil
}

func (u *RekonsiliasiUseCase) UpdateRekonsiliasi(ctx context.Context, idRekonsiliasi int32, userID int32, req model.UpdateRekonsiliasiRequest) (*model.RekonsiliasiDetailResponse, error) {
	if len(req.MaterialRows) == 0 {
		return nil, ErrRekonsiliasiValidation
	}

	existingRows, err := u.repo.ListRekonsiliasiMaterialRowsByRekonsiliasiID(ctx, idRekonsiliasi)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRekonsiliasiNotFound
		}
		return nil, fmt.Errorf("%w: failed to get rekonsiliasi rows", ErrRekonsiliasiUnavailable)
	}
	if len(existingRows) == 0 {
		if _, err := u.repo.GetRekonsiliasiByID(ctx, idRekonsiliasi); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrRekonsiliasiNotFound
			}
			return nil, fmt.Errorf("%w: failed to validate rekonsiliasi", ErrRekonsiliasiUnavailable)
		}
	}

	rowSet := make(map[int32]struct{}, len(existingRows))
	for _, row := range existingRows {
		rowSet[row.IDRekonsiliasiMaterialRow] = struct{}{}
	}

	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to start transaction", ErrRekonsiliasiUnavailable)
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()

	qtx := entity.New(tx)
	for _, rowReq := range req.MaterialRows {
		if _, ok := rowSet[rowReq.IDRekonsiliasiMaterialRow]; !ok {
			return nil, ErrRekonsiliasiValidation
		}

		if _, err := qtx.UpdateRekonsiliasiMaterialRowManualFields(ctx, entity.UpdateRekonsiliasiMaterialRowManualFieldsParams{
			RatioInput:               mustNumeric(rowReq.RatioInput),
			QtyPerPcsInput:           mustNumeric(rowReq.QtyPerPcsInput),
			QtyActualKirimManual:     rowReq.QtyActualKirimManual,
			RejectQty:                rowReq.RejectQty,
			ReturQty:                 rowReq.ReturQty,
			Keterangan:               rowReq.Keterangan,
			IDRekonsiliasiMaterialRow: rowReq.IDRekonsiliasiMaterialRow,
		}); err != nil {
			return nil, fmt.Errorf("%w: failed to update rekonsiliasi material row", ErrRekonsiliasiUnavailable)
		}

		if err := qtx.DeleteRekonsiliasiTerimaEntriesByRowID(ctx, rowReq.IDRekonsiliasiMaterialRow); err != nil {
			return nil, fmt.Errorf("%w: failed to replace rekonsiliasi terima entries", ErrRekonsiliasiUnavailable)
		}

		for _, entryReq := range rowReq.TerimaEntries {
			if _, err := qtx.CreateRekonsiliasiTerimaEntry(ctx, entity.CreateRekonsiliasiTerimaEntryParams{
				IDRekonsiliasiMaterialRow: rowReq.IDRekonsiliasiMaterialRow,
				EntryType:                 entryReq.EntryType,
				EntryLabel:                entryReq.EntryLabel,
				Qty:                       entryReq.Qty,
				Note:                      entryReq.Note,
			}); err != nil {
				return nil, fmt.Errorf("%w: failed to create rekonsiliasi terima entry", ErrRekonsiliasiUnavailable)
			}
		}
	}

	if _, err := qtx.TouchRekonsiliasi(ctx, entity.TouchRekonsiliasiParams{
		UpdatedBy:      pgtype.Int4{Int32: userID, Valid: true},
		IDRekonsiliasi: idRekonsiliasi,
	}); err != nil {
		return nil, fmt.Errorf("%w: failed to touch rekonsiliasi", ErrRekonsiliasiUnavailable)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit rekonsiliasi update", ErrRekonsiliasiUnavailable)
	}

	return u.GetRekonsiliasi(ctx, idRekonsiliasi)
}

func (u *RekonsiliasiUseCase) RefreshRekonsiliasi(ctx context.Context, idRekonsiliasi int32, userID int32) (*model.RekonsiliasiDetailResponse, error) {
	current, err := u.repo.GetRekonsiliasiByID(ctx, idRekonsiliasi)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRekonsiliasiNotFound
		}
		return nil, fmt.Errorf("%w: failed to get rekonsiliasi", ErrRekonsiliasiUnavailable)
	}

	existingRows, err := u.repo.ListRekonsiliasiMaterialRowsByRekonsiliasiID(ctx, idRekonsiliasi)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get rekonsiliasi rows", ErrRekonsiliasiUnavailable)
	}
	existingEntries, err := u.repo.ListRekonsiliasiTerimaEntriesByRekonsiliasiID(ctx, idRekonsiliasi)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get rekonsiliasi terima entries", ErrRekonsiliasiUnavailable)
	}

	manualState := captureRekonsiliasiManualState(existingRows, existingEntries)

	snapshot, err := u.buildSnapshot(ctx, current.IDWo)
	if err != nil {
		return nil, err
	}

	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to start transaction", ErrRekonsiliasiUnavailable)
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()

	qtx := entity.New(tx)
	if _, err := qtx.UpdateRekonsiliasiSnapshot(ctx, entity.UpdateRekonsiliasiSnapshotParams{
		Jasa:             snapshot.header.Jasa,
		NoPo:             snapshot.header.NoPo,
		Delivery:         snapshot.header.Delivery,
		Buyer:            snapshot.header.Buyer,
		Brand:            snapshot.header.Brand,
		Style:            snapshot.header.Style,
		QtyPo:            snapshot.header.QtyPo,
		PlanCutTotal:     snapshot.header.PlanCutTotal,
		ConsBajuSummary:  snapshot.consSummary,
		NamaBahan:        snapshot.namaBahan,
		WarnaKainSummary: snapshot.warnaKain,
		UpdatedBy:        pgtype.Int4{Int32: userID, Valid: true},
		IDRekonsiliasi:   idRekonsiliasi,
	}); err != nil {
		return nil, fmt.Errorf("%w: failed to refresh rekonsiliasi header", ErrRekonsiliasiUnavailable)
	}

	if err := qtx.DeleteRekonsiliasiMaterialRowsByRekonsiliasiID(ctx, idRekonsiliasi); err != nil {
		return nil, fmt.Errorf("%w: failed to rebuild rekonsiliasi rows", ErrRekonsiliasiUnavailable)
	}

	for _, sourceRow := range snapshot.sourceRows {
		state, ok := manualState[seedRowKey(sourceRow)]
		if !ok {
			state = rekonsiliasiManualState{
				RatioInput:           sourceRow.RatioSource,
				QtyPerPcsInput:       0,
				QtyActualKirimManual: 0,
				RejectQty:            0,
				ReturQty:             0,
				Keterangan:           "",
				TerimaEntries:        nil,
			}
		}

		createdRow, err := qtx.CreateRekonsiliasiMaterialRow(ctx, entity.CreateRekonsiliasiMaterialRowParams{
			IDRekonsiliasi:        idRekonsiliasi,
			RowNo:                 sourceRow.RowNo,
			Kategori:              sourceRow.Kategori,
			Description:           sourceRow.Description,
			SizeLabel:             sourceRow.SizeLabel,
			RatioSource:           mustNumeric(sourceRow.RatioSource),
			RatioInput:            mustNumeric(state.RatioInput),
			QtyPerPcsInput:        mustNumeric(state.QtyPerPcsInput),
			QtyWo:                 sourceRow.QtyWO,
			Toleransi:             sourceRow.Toleransi,
			Satuan:                sourceRow.Satuan,
			QtyActualKirimSource:  sourceRow.QtyActualKirimSource,
			QtyActualKirimManual:  state.QtyActualKirimManual,
			RejectQty:             state.RejectQty,
			ReturQty:              state.ReturQty,
			Keterangan:            state.Keterangan,
			IDMaterialListItem:    nullableInt32Param(sourceRow.IDMaterialListItem),
			IDWoShell:             nullableInt32Param(sourceRow.IDWoShell),
			IDWoTrim:              nullableInt32Param(sourceRow.IDWoTrim),
		})
		if err != nil {
			return nil, fmt.Errorf("%w: failed to recreate rekonsiliasi row", ErrRekonsiliasiUnavailable)
		}

		for _, entry := range state.TerimaEntries {
			if _, err := qtx.CreateRekonsiliasiTerimaEntry(ctx, entity.CreateRekonsiliasiTerimaEntryParams{
				IDRekonsiliasiMaterialRow: createdRow.IDRekonsiliasiMaterialRow,
				EntryType:                 entry.EntryType,
				EntryLabel:                entry.EntryLabel,
				Qty:                       entry.Qty,
				Note:                      entry.Note,
			}); err != nil {
				return nil, fmt.Errorf("%w: failed to recreate rekonsiliasi terima entry", ErrRekonsiliasiUnavailable)
			}
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit rekonsiliasi refresh", ErrRekonsiliasiUnavailable)
	}

	return u.GetRekonsiliasi(ctx, idRekonsiliasi)
}

func (u *RekonsiliasiUseCase) buildSnapshot(ctx context.Context, idWo int32) (*rekonsiliasiSnapshot, error) {
	header, err := u.repo.GetRekonsiliasiSourceHeader(ctx, idWo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRekonsiliasiSourceNotFound
		}
		return nil, fmt.Errorf("%w: failed to load rekonsiliasi source header", ErrRekonsiliasiUnavailable)
	}

	shellRows, err := u.repo.ListRekonsiliasiShellSourcesByWO(ctx, idWo)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to load rekonsiliasi shell sources", ErrRekonsiliasiUnavailable)
	}

	colorRows, err := u.repo.ListRekonsiliasiColorSourcesByWO(ctx, idWo)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to load rekonsiliasi color sources", ErrRekonsiliasiUnavailable)
	}

	materialRows, err := u.repo.ListRekonsiliasiMaterialSourceRowsByWO(ctx, idWo)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to load rekonsiliasi material sources", ErrRekonsiliasiUnavailable)
	}

	warnaSummary := extractDistinctWarna(shellRows, colorRows)
	sourceRows := make([]rekonsiliasiMaterialSeed, 0, len(materialRows))
	for idx, row := range materialRows {
		sourceRows = append(sourceRows, rekonsiliasiMaterialSeed{
			RowNo:                int32(idx + 1),
			Kategori:             row.Kategori,
			Description:          interfaceToString(row.Description),
			SizeLabel:            row.SizeLabel,
			RatioSource:          numericToFloat64(row.RatioSource),
			QtyWO:                interfaceToInt32(row.QtyWo),
			Toleransi:            row.Toleransi,
			Satuan:               row.Satuan,
			QtyActualKirimSource: interfaceToInt32(row.QtyActualKirimSource),
			IDMaterialListItem:   int32Ptr(row.IDMaterialListItem),
			IDWoShell:            nullableInt32Ptr(row.IDWoShell),
			IDWoTrim:             nullableInt32Ptr(row.IDWoTrim),
		})
	}

	return &rekonsiliasiSnapshot{
		header:         header,
		consSummary:    buildConsBajuSummary(shellRows),
		namaBahan:      buildNamaBahanSummary(shellRows),
		warnaKain:      warnaSummary,
		colorSummaries: buildRekonsiliasiColorSummaries(colorRows),
		sourceRows:     sourceRows,
	}, nil
}

func buildRekonsiliasiColorSummaries(rows []entity.ListRekonsiliasiColorSourcesByWORow) []model.RekonsiliasiColorSummaryResponse {
	type group struct {
		Color         string
		QtyOrder      int32
		QtyKirim      int32
		Balance       int32
		SizeBreakdown []model.RekonsiliasiColorSizeSummaryResponse
	}

	groupMap := make(map[string]*group)
	order := make([]string, 0)
	for _, row := range rows {
		item, ok := groupMap[row.Color]
		if !ok {
			item = &group{Color: row.Color}
			groupMap[row.Color] = item
			order = append(order, row.Color)
		}

		balance := row.QtyOrder - row.QtyKirim
		item.QtyOrder += row.QtyOrder
		item.QtyKirim += row.QtyKirim
		item.Balance += balance
		item.SizeBreakdown = append(item.SizeBreakdown, model.RekonsiliasiColorSizeSummaryResponse{
			Size:     row.Size,
			QtyOrder: row.QtyOrder,
			QtyKirim: row.QtyKirim,
			Balance:  balance,
		})
	}

	result := make([]model.RekonsiliasiColorSummaryResponse, 0, len(order))
	for _, color := range order {
		item := groupMap[color]
		result = append(result, model.RekonsiliasiColorSummaryResponse{
			Color:         item.Color,
			QtyOrder:      item.QtyOrder,
			QtyKirim:      item.QtyKirim,
			Balance:       item.Balance,
			SizeBreakdown: item.SizeBreakdown,
		})
	}
	return result
}

func buildRekonsiliasiMaterialRows(rows []entity.RekonsiliasiMaterialRow, entries []entity.RekonsiliasiTerimaEntry) []model.RekonsiliasiMaterialRowResponse {
	entryMap := make(map[int32][]model.RekonsiliasiTerimaEntryResponse)
	totalTerimaMap := make(map[int32]int32)
	for _, entry := range entries {
		sign := int32(1)
		if entry.EntryType == "untuk" {
			sign = -1
		}
		totalTerimaMap[entry.IDRekonsiliasiMaterialRow] += sign * entry.Qty
		entryMap[entry.IDRekonsiliasiMaterialRow] = append(entryMap[entry.IDRekonsiliasiMaterialRow], model.RekonsiliasiTerimaEntryResponse{
			IDRekonsiliasiTerimaEntry: entry.IDRekonsiliasiTerimaEntry,
			EntryType:                 entry.EntryType,
			EntryLabel:                entry.EntryLabel,
			Qty:                       entry.Qty,
			Note:                      entry.Note,
			CreatedAt:                 entry.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:                 entry.UpdatedAt.Time.Format(time.RFC3339),
		})
	}

	result := make([]model.RekonsiliasiMaterialRowResponse, 0, len(rows))
	for _, row := range rows {
		qtyActualKirim := row.QtyActualKirimSource + row.QtyActualKirimManual
		totalTerima := totalTerimaMap[row.IDRekonsiliasiMaterialRow]
		qtyPerPcsInput := numericToFloat64(row.QtyPerPcsInput)
		consActual := float64(totalTerima) - (float64(qtyActualKirim) * qtyPerPcsInput)
		balance := float64(totalTerima) - consActual
		lastBalance := balance - float64(row.RejectQty) - float64(row.ReturQty)

		result = append(result, model.RekonsiliasiMaterialRowResponse{
			IDRekonsiliasiMaterialRow: row.IDRekonsiliasiMaterialRow,
			RowNo:                     row.RowNo,
			Kategori:                  row.Kategori,
			Description:               row.Description,
			SizeLabel:                 row.SizeLabel,
			RatioSource:               numericToFloat64(row.RatioSource),
			RatioInput:                numericToFloat64(row.RatioInput),
			QtyPerPcsInput:            qtyPerPcsInput,
			QtyWO:                     row.QtyWo,
			Toleransi:                 row.Toleransi,
			Satuan:                    row.Satuan,
			QtyActualKirimSource:      row.QtyActualKirimSource,
			QtyActualKirimManual:      row.QtyActualKirimManual,
			QtyActualKirim:            qtyActualKirim,
			TotalTerima:               totalTerima,
			ConsActual:                consActual,
			Balance:                   balance,
			RejectQty:                 row.RejectQty,
			ReturQty:                  row.ReturQty,
			LastBalance:               lastBalance,
			Keterangan:                row.Keterangan,
			IDMaterialListItem:        nullableInt32Ptr(row.IDMaterialListItem),
			IDWoShell:                 nullableInt32Ptr(row.IDWoShell),
			IDWoTrim:                  nullableInt32Ptr(row.IDWoTrim),
			TerimaEntries:             entryMap[row.IDRekonsiliasiMaterialRow],
		})
	}

	return result
}

func captureRekonsiliasiManualState(rows []entity.RekonsiliasiMaterialRow, entries []entity.RekonsiliasiTerimaEntry) map[string]rekonsiliasiManualState {
	entryMap := make(map[int32][]model.UpdateRekonsiliasiTerimaEntryRequest)
	for _, entry := range entries {
		entryMap[entry.IDRekonsiliasiMaterialRow] = append(entryMap[entry.IDRekonsiliasiMaterialRow], model.UpdateRekonsiliasiTerimaEntryRequest{
			IDRekonsiliasiTerimaEntry: &entry.IDRekonsiliasiTerimaEntry,
			EntryType:                 entry.EntryType,
			EntryLabel:                entry.EntryLabel,
			Qty:                       entry.Qty,
			Note:                      entry.Note,
		})
	}

	state := make(map[string]rekonsiliasiManualState, len(rows))
	for _, row := range rows {
		state[rowKey(row)] = rekonsiliasiManualState{
			RatioInput:           numericToFloat64(row.RatioInput),
			QtyPerPcsInput:       numericToFloat64(row.QtyPerPcsInput),
			QtyActualKirimManual: row.QtyActualKirimManual,
			RejectQty:            row.RejectQty,
			ReturQty:             row.ReturQty,
			Keterangan:           row.Keterangan,
			TerimaEntries:        entryMap[row.IDRekonsiliasiMaterialRow],
		}
	}
	return state
}

func rowKey(row entity.RekonsiliasiMaterialRow) string {
	if row.IDMaterialListItem.Valid {
		return fmt.Sprintf("mli:%d", row.IDMaterialListItem.Int32)
	}
	if row.IDWoShell.Valid {
		return fmt.Sprintf("shell:%d:%s:%s", row.IDWoShell.Int32, row.Description, row.SizeLabel)
	}
	if row.IDWoTrim.Valid {
		return fmt.Sprintf("trim:%d:%s:%s", row.IDWoTrim.Int32, row.Description, row.SizeLabel)
	}
	return fmt.Sprintf("row:%d:%s:%s", row.RowNo, row.Description, row.SizeLabel)
}

func seedRowKey(row rekonsiliasiMaterialSeed) string {
	if row.IDMaterialListItem != nil {
		return fmt.Sprintf("mli:%d", *row.IDMaterialListItem)
	}
	if row.IDWoShell != nil {
		return fmt.Sprintf("shell:%d:%s:%s", *row.IDWoShell, row.Description, row.SizeLabel)
	}
	if row.IDWoTrim != nil {
		return fmt.Sprintf("trim:%d:%s:%s", *row.IDWoTrim, row.Description, row.SizeLabel)
	}
	return fmt.Sprintf("row:%d:%s:%s", row.RowNo, row.Description, row.SizeLabel)
}

func buildConsBajuSummary(rows []entity.ListRekonsiliasiShellSourcesByWORow) string {
	seen := make(map[string]struct{})
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		value := formatCompactFloat(numericToFloat64(row.Cons))
		if value == "" || value == "0" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return strings.Join(values, "/")
}

func buildNamaBahanSummary(rows []entity.ListRekonsiliasiShellSourcesByWORow) string {
	seen := make(map[string]struct{})
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		value := strings.TrimSpace(row.Deskripsi)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return strings.Join(values, " / ")
}

func extractDistinctWarna(shellRows []entity.ListRekonsiliasiShellSourcesByWORow, colorRows []entity.ListRekonsiliasiColorSourcesByWORow) []string {
	seen := make(map[string]struct{})
	values := make([]string, 0, len(shellRows)+len(colorRows))
	for _, row := range shellRows {
		value := strings.TrimSpace(row.Color)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	for _, row := range colorRows {
		value := strings.TrimSpace(row.Color)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func decodeWarnaSummary(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}

	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func interfaceToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprint(v)
	}
}

func interfaceToInt32(value interface{}) int32 {
	switch v := value.(type) {
	case int32:
		return v
	case int64:
		return int32(v)
	case float64:
		return int32(v)
	case []byte:
		var parsed int64
		fmt.Sscan(string(v), &parsed)
		return int32(parsed)
	case string:
		var parsed int64
		fmt.Sscan(v, &parsed)
		return int32(parsed)
	default:
		return 0
	}
}

func int32Ptr(value int32) *int32 {
	return &value
}

func formatCompactFloat(value float64) string {
	formatted := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", value), "0"), ".")
	if formatted == "" {
		return "0"
	}
	return formatted
}

func mapRekonsiliasiDBError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return ErrRekonsiliasiSourceNotFound
		case "23505":
			if strings.Contains(strings.ToLower(pgErr.ConstraintName), "id_wo") {
				return ErrRekonsiliasiAlreadyExists
			}
			return ErrRekonsiliasiAlreadyExists
		}
	}

	return fmt.Errorf("%w: %v", ErrRekonsiliasiUnavailable, err)
}
