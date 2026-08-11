package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

var (
	// ErrMaterialListExportMarkerPlanConflict maps directly to the stable
	// marker_plan_conflict code for a future HTTP 409 response.
	ErrMaterialListExportMarkerPlanConflict = errors.New("marker_plan_conflict")
	ErrMaterialListExportDataConflict       = errors.New("material_list_export_data_conflict")
	ErrMaterialListExportUnavailable        = errors.New("material list export unavailable")
)

type materialListExportRepository interface {
	GetMaterialListExportHeader(context.Context, int32) (entity.GetMaterialListExportHeaderRow, error)
	ListMaterialListExportItems(context.Context, int32) ([]entity.ListMaterialListExportItemsRow, error)
	ListMaterialListExportShellSizes(context.Context, int32) ([]entity.ListMaterialListExportShellSizesRow, error)
	ListMaterialListExportMarkerPlans(context.Context, int32) ([]entity.ListMaterialListExportMarkerPlansRow, error)
	ListMaterialListExportMarkerRatios(context.Context, int32) ([]entity.ListMaterialListExportMarkerRatiosRow, error)
	ListMaterialListExportCutting(context.Context, int32) ([]entity.ListMaterialListExportCuttingRow, error)
	ListMaterialListExportSuratJalanByDate(context.Context, int32) ([]entity.ListMaterialListExportSuratJalanByDateRow, error)
	ListMaterialListExportReceivedByDate(context.Context, int32) ([]entity.ListMaterialListExportReceivedByDateRow, error)
	ListMaterialListExportReconciliationRows(context.Context, entity.ListMaterialListExportReconciliationRowsParams) ([]entity.ListMaterialListExportReconciliationRowsRow, error)
}

// MaterialListExportUseCase reads a fixed, batched snapshot and composes it
// into the renderer-neutral MaterialListExport model.
type MaterialListExportUseCase struct{ repo materialListExportRepository }

func NewMaterialListExportUseCase(repo materialListExportRepository) (*MaterialListExportUseCase, error) {
	if repo == nil {
		return nil, errors.New("material list export repository is required")
	}
	return &MaterialListExportUseCase{repo: repo}, nil
}

func (u *MaterialListExportUseCase) Compose(ctx context.Context, idMaterialList int32) (*model.MaterialListExport, error) {
	header, err := u.repo.GetMaterialListExportHeader(ctx, idMaterialList)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMaterialListNotFound
		}
		return nil, fmt.Errorf("%w: header: %v", ErrMaterialListExportUnavailable, err)
	}
	items, err := u.repo.ListMaterialListExportItems(ctx, idMaterialList)
	if err != nil {
		return nil, materialListExportReadError("items", err)
	}
	shellSizes, err := u.repo.ListMaterialListExportShellSizes(ctx, header.IDWo)
	if err != nil {
		return nil, materialListExportReadError("shell sizes", err)
	}
	plans, err := u.repo.ListMaterialListExportMarkerPlans(ctx, header.IDWo)
	if err != nil {
		return nil, materialListExportReadError("marker plans", err)
	}
	ratios, err := u.repo.ListMaterialListExportMarkerRatios(ctx, header.IDWo)
	if err != nil {
		return nil, materialListExportReadError("marker ratios", err)
	}
	cutting, err := u.repo.ListMaterialListExportCutting(ctx, header.IDWo)
	if err != nil {
		return nil, materialListExportReadError("cutting", err)
	}
	sj, err := u.repo.ListMaterialListExportSuratJalanByDate(ctx, idMaterialList)
	if err != nil {
		return nil, materialListExportReadError("surat jalan", err)
	}
	received, err := u.repo.ListMaterialListExportReceivedByDate(ctx, idMaterialList)
	if err != nil {
		return nil, materialListExportReadError("received", err)
	}
	reconciliation, err := u.repo.ListMaterialListExportReconciliationRows(ctx, entity.ListMaterialListExportReconciliationRowsParams{IDWo: header.IDWo, IDMaterialList: idMaterialList})
	if err != nil {
		return nil, materialListExportReadError("reconciliation", err)
	}

	return composeMaterialListExport(materialListExportSnapshot{header, items, shellSizes, plans, ratios, cutting, sj, received, reconciliation})
}

func materialListExportReadError(part string, err error) error {
	return fmt.Errorf("%w: %s: %v", ErrMaterialListExportUnavailable, part, err)
}

type materialListExportSnapshot struct {
	header         entity.GetMaterialListExportHeaderRow
	items          []entity.ListMaterialListExportItemsRow
	shellSizes     []entity.ListMaterialListExportShellSizesRow
	plans          []entity.ListMaterialListExportMarkerPlansRow
	ratios         []entity.ListMaterialListExportMarkerRatiosRow
	cutting        []entity.ListMaterialListExportCuttingRow
	sj             []entity.ListMaterialListExportSuratJalanByDateRow
	received       []entity.ListMaterialListExportReceivedByDateRow
	reconciliation []entity.ListMaterialListExportReconciliationRowsRow
}

type exportShell struct {
	id                 int32
	color, description string
	sizes              map[int32]exportShellSize
}
type exportShellSize struct {
	id, sizeID int32
	name       string
	order      int64
}
type exportPlan struct {
	known bool
	qty   int64
}

func composeMaterialListExport(s materialListExportSnapshot) (*model.MaterialListExport, error) {
	shells, sizes, sizeOrder, err := materialListExportShells(s.shellSizes)
	if err != nil {
		return nil, err
	}
	planByShellSize, err := materialListExportPlans(shells, s.plans, s.ratios)
	if err != nil {
		return nil, err
	}
	actualByShellSize := make(map[int32]int64, len(s.cutting))
	for _, row := range s.cutting {
		actualByShellSize[row.IDWoShellSize] = row.ActualCutQty
	}

	export := &model.MaterialListExport{
		Header:    model.MaterialListExportHeader{MaterialListID: s.header.IDMaterialList, MaterialListName: s.header.MaterialListName, WorkOrderID: s.header.IDWo, Buyer: s.header.Buyer, Model: s.header.Model, Style: s.header.Style, WorkOrderQty: int64(s.header.WoQty), FobCmt: s.header.FobCmt, Delivery: s.header.Delivery.Time},
		DetailQty: materialListExportDetail(shells, sizes, sizeOrder, planByShellSize, actualByShellSize),
	}
	sjByItem, sjDates := materialListExportTransactionsSJ(s.sj)
	receivedByItem, receivedDates := materialListExportTransactionsReceived(s.received)
	export.SJDates = sjDates
	export.ReceivedDates = receivedDates
	reconciliationByItem := materialListExportReconciliation(s.reconciliation)
	for _, item := range s.items {
		export.MaterialRows = append(export.MaterialRows, materialListExportRow(item, shells, planByShellSize, actualByShellSize, sjByItem[item.IDMaterialListItem], receivedByItem[item.IDMaterialListItem], reconciliationByItem[item.IDMaterialListItem], int64(s.header.WoQty)))
	}
	return export, nil
}

func materialListExportShells(rows []entity.ListMaterialListExportShellSizesRow) (map[int32]exportShell, map[int32]model.MaterialListExportSize, []int32, error) {
	shells := map[int32]exportShell{}
	sizes := map[int32]model.MaterialListExportSize{}
	ratioState := map[int32]struct {
		value                  int32
		available, unavailable bool
	}{}
	var order []int32
	for _, row := range rows {
		shell := shells[row.IDWoShell]
		if shell.sizes == nil {
			shell = exportShell{id: row.IDWoShell, color: row.Color, description: row.ShellDescription, sizes: map[int32]exportShellSize{}}
		}
		if !row.IDWoShellSize.Valid || !row.IDSize.Valid {
			shells[row.IDWoShell] = shell
			continue
		}
		shell.sizes[row.IDSize.Int32] = exportShellSize{id: row.IDWoShellSize.Int32, sizeID: row.IDSize.Int32, name: row.NamaSize.String, order: int64(row.OrderQty.Int32)}
		shells[row.IDWoShell] = shell
		state := ratioState[row.IDSize.Int32]
		if row.ShellSizeRatio.Valid {
			if state.available && state.value != row.ShellSizeRatio.Int32 {
				return nil, nil, nil, fmt.Errorf("%w: conflicting work order shell size ratio for size %d", ErrMaterialListExportDataConflict, row.IDSize.Int32)
			}
			state.value, state.available = row.ShellSizeRatio.Int32, true
		} else {
			state.unavailable = true
		}
		ratioState[row.IDSize.Int32] = state
		if _, ok := sizes[row.IDSize.Int32]; !ok {
			sizes[row.IDSize.Int32] = model.MaterialListExportSize{ID: row.IDSize.Int32, Name: row.NamaSize.String}
			order = append(order, row.IDSize.Int32)
		}
	}
	for id, state := range ratioState {
		if state.available && !state.unavailable {
			value := state.value
			size := sizes[id]
			size.Ratio = &value
			sizes[id] = size
		}
	}
	sort.Slice(order, func(i, j int) bool { return sizes[order[i]].Name < sizes[order[j]].Name })
	return shells, sizes, order, nil
}

func materialListExportPlans(shells map[int32]exportShell, parents []entity.ListMaterialListExportMarkerPlansRow, rows []entity.ListMaterialListExportMarkerRatiosRow) (map[int32]exportPlan, error) {
	parentsByShell := map[int32][]int32{}
	for _, parent := range parents {
		if _, found := shells[parent.IDWoShell]; !found {
			return nil, fmt.Errorf("%w: marker plan shell is outside work order", ErrMaterialListExportDataConflict)
		}
		parentsByShell[parent.IDWoShell] = append(parentsByShell[parent.IDWoShell], parent.IDMarkerPlan)
	}
	for shellID, parentIDs := range parentsByShell {
		if len(parentIDs) > 1 {
			return nil, fmt.Errorf("%w: shell %d", ErrMaterialListExportMarkerPlanConflict, shellID)
		}
	}
	plans := map[int32]exportPlan{}
	parentShell := map[int32]int32{}
	for shellID, parentIDs := range parentsByShell {
		if len(parentIDs) == 1 {
			parentShell[parentIDs[0]] = shellID
		}
	}
	for _, row := range rows {
		markerPlanShell, found := parentShell[row.IDMarkerPlan]
		if !found || markerPlanShell != row.MarkerPlanShellID {
			return nil, fmt.Errorf("%w: marker ratio does not match marker plan", ErrMaterialListExportDataConflict)
		}
		if row.RatioShellID != row.RatioSizeShellID {
			return nil, fmt.Errorf("%w: marker ratio size shell does not match ratio shell", ErrMaterialListExportDataConflict)
		}
		shell, found := shells[row.RatioShellID]
		if !found {
			return nil, fmt.Errorf("%w: marker ratio shell is outside work order", ErrMaterialListExportDataConflict)
		}
		size, found := shell.sizes[row.IDSize]
		if !found || size.id != row.IDWoShellSize {
			return nil, fmt.Errorf("%w: ratio size does not match ratio shell", ErrMaterialListExportDataConflict)
		}
		plan, found := plans[row.IDWoShellSize]
		if !found {
			plan.known = true
		}
		spread := numericToFloat64Ptr(row.PlanSpreadingGelaran)
		if spread == nil {
			return nil, fmt.Errorf("%w: marker plan spreading is unavailable", ErrMaterialListExportDataConflict)
		}
		plan.qty += int64(math.Round(float64(row.RatioPlan) * *spread))
		plans[row.IDWoShellSize] = plan
	}
	return plans, nil
}

func materialListExportDetail(shells map[int32]exportShell, sizes map[int32]model.MaterialListExportSize, sizeOrder []int32, plans map[int32]exportPlan, actual map[int32]int64) model.MaterialListExportDetailQty {
	detail := model.MaterialListExportDetailQty{}
	for _, id := range sizeOrder {
		detail.Sizes = append(detail.Sizes, sizes[id])
	}
	shellIDs := sortedShellIDs(shells)
	for _, shellID := range shellIDs {
		shell := shells[shellID]
		out := model.MaterialListExportShellQty{ShellID: shell.id, Color: shell.color, Description: shell.description}
		for _, sizeID := range sizeOrder {
			cell := model.MaterialListExportShellSizeQty{SizeID: sizeID}
			if size, ok := shell.sizes[sizeID]; ok {
				cell.Order = int64Ptr(size.order)
				if plan, ok := plans[size.id]; ok && plan.known {
					cell.MarkerPlan = int64Ptr(plan.qty)
				}
				if qty, ok := actual[size.id]; ok {
					cell.ActualCut = int64Ptr(qty)
				}
			}
			out.Values = append(out.Values, cell)
		}
		detail.Shells = append(detail.Shells, out)
	}
	for _, sizeID := range sizeOrder {
		var order, rms, cut int64
		anyOrder, anyRMS, anyCut, completeRMS, completeCut := false, false, false, true, true
		for _, shell := range detail.Shells {
			cell := shell.Values[indexForSize(sizeOrder, sizeID)]
			if cell.Order != nil {
				order += *cell.Order
				anyOrder = true
				if cell.MarkerPlan != nil {
					rms += *cell.MarkerPlan
					anyRMS = true
				} else {
					completeRMS = false
				}
				if cell.ActualCut != nil {
					cut += *cell.ActualCut
					anyCut = true
				} else {
					completeCut = false
				}
			}
		}
		summary := model.MaterialListExportSizeSummary{SizeID: sizeID}
		if anyOrder {
			summary.OrderTotal = int64Ptr(order)
		}
		if anyRMS {
			summary.RMSTotal = int64Ptr(rms)
		}
		if anyCut {
			summary.TotalActCut = int64Ptr(cut)
		}
		if anyOrder && completeRMS && completeCut {
			summary.Balance = int64Ptr(cut - rms)
		}
		detail.Summaries = append(detail.Summaries, summary)
	}
	return detail
}

func materialListExportRow(item entity.ListMaterialListExportItemsRow, shells map[int32]exportShell, plans map[int32]exportPlan, actual map[int32]int64, sj, received map[time.Time]int64, reconciliation *entity.ListMaterialListExportReconciliationRowsRow, wholeWOQty int64) model.MaterialListExportRow {
	row := model.MaterialListExportRow{MaterialListItemID: item.IDMaterialListItem, Item: item.Item, Description: item.Description, Category: nullableStringPtr(item.Category), Unit: item.Unit, ConsPerPC: numericToFloat64Ptr(item.ConsPerPc), QtyWoScope: nullableStringPtr(item.QtyWoScope), SJByDate: sj, ReceivedByDate: received, CuttingSource: model.MaterialListCuttingNone}
	row.Applicability = materialListExportApplicability(item, shells)
	row.Source = materialListExportSource(item)
	selected, deterministic := materialListExportScope(item, shells, wholeWOQty)
	if deterministic {
		row.QtyWO = materialListExportOrderTotal(selected)
	}
	if row.QtyWO != nil && row.ConsPerPC != nil {
		row.Total = exportFloat64Ptr(float64(*row.QtyWO) * *row.ConsPerPC)
	}
	if deterministic {
		row.CuttingQty, row.CuttingSource = materialListExportCutting(selected, plans, actual)
	}
	if row.CuttingQty != nil && row.ConsPerPC != nil && row.Source.AllowancePercent != nil {
		row.ActualConsumption = exportFloat64Ptr(float64(*row.CuttingQty) * *row.ConsPerPC * (1 + float64(*row.Source.AllowancePercent)/100))
	}
	if len(sj) > 0 {
		row.SJTotal = int64Ptr(sumTransactions(sj))
	}
	if len(received) > 0 {
		row.ReceivedTotal = int64Ptr(sumTransactions(received))
	}
	if row.SJTotal != nil && row.ReceivedTotal != nil {
		row.BalanceReceived = int64Ptr(*row.ReceivedTotal - *row.SJTotal)
	}
	if reconciliation != nil {
		row.RejectRetur = int64Ptr(int64(reconciliation.RejectQty) + int64(reconciliation.ReturQty))
		if reconciliation.Keterangan != "" {
			note := reconciliation.Keterangan
			row.ReconciliationNote = &note
		}
	}
	if row.ReceivedTotal != nil && row.ActualConsumption != nil && row.RejectRetur != nil {
		row.FinalBalance = exportFloat64Ptr(float64(*row.ReceivedTotal) - (*row.ActualConsumption + float64(*row.RejectRetur)))
	}
	return row
}

func materialListExportApplicability(item entity.ListMaterialListExportItemsRow, shells map[int32]exportShell) model.MaterialListExportApplicability {
	app := model.MaterialListExportApplicability{ShellID: nullableInt32Ptr(item.IDQtyWoShell), SizeID: nullableInt32Ptr(item.IDQtyWoSize)}
	if app.ShellID != nil {
		if shell, ok := shells[*app.ShellID]; ok {
			color := shell.color
			app.ShellColor = &color
		}
	}
	if app.SizeID != nil {
		for _, shell := range shells {
			if size, ok := shell.sizes[*app.SizeID]; ok {
				name := size.name
				app.SizeName = &name
				break
			}
		}
	}
	return app
}

func materialListExportSource(item entity.ListMaterialListExportItemsRow) model.MaterialListExportMaterialSource {
	source := model.MaterialListExportMaterialSource{
		ShellID:          nullableInt32Ptr(item.SourceShellID),
		ShellDescription: nullableStringPtr(item.SourceShellDescription),
		ShellColor:       nullableStringPtr(item.SourceShellColor),
		TrimID:           nullableInt32Ptr(item.SourceTrimID),
		TrimItem:         nullableStringPtr(item.SourceTrimItem),
		TrimDescription:  nullableStringPtr(item.SourceTrimDescription),
		TrimColor:        nullableStringPtr(item.SourceTrimColor),
	}
	// A dual source has no established allowance precedence and stays unavailable.
	if source.ShellID != nil && source.TrimID == nil && item.SourceShellAllow.Valid {
		value := item.SourceShellAllow.Int32
		source.AllowancePercent = &value
	}
	if source.TrimID != nil && source.ShellID == nil && item.SourceTrimAllow.Valid {
		value := item.SourceTrimAllow.Int32
		source.AllowancePercent = &value
	}
	return source
}

func materialListExportScope(item entity.ListMaterialListExportItemsRow, shells map[int32]exportShell, wholeWOQty int64) ([]exportShellSize, bool) {
	if !item.QtyWoScope.Valid {
		return nil, false
	}
	var selected []exportShellSize
	switch item.QtyWoScope.String {
	case qtyWoScopeWholeWO:
		if item.IDQtyWoShell.Valid || item.IDQtyWoSize.Valid {
			return nil, false
		}
		for _, shell := range shells {
			for _, size := range shell.sizes {
				selected = append(selected, size)
			}
		}
		// WHOLE_WO QTY is deliberately not derived from selected shell-size rows.
		return append([]exportShellSize{{id: -1, order: wholeWOQty}}, selected...), true
	case qtyWoScopeSize:
		if item.IDQtyWoShell.Valid || !item.IDQtyWoSize.Valid {
			return nil, false
		}
		for _, shell := range shells {
			if size, ok := shell.sizes[item.IDQtyWoSize.Int32]; ok {
				selected = append(selected, size)
			}
		}
	case qtyWoScopeColor:
		if !item.IDQtyWoShell.Valid || item.IDQtyWoSize.Valid {
			return nil, false
		}
		shell, ok := shells[item.IDQtyWoShell.Int32]
		if !ok {
			return nil, false
		}
		for _, size := range shell.sizes {
			selected = append(selected, size)
		}
	case qtyWoScopeColorSize:
		if !item.IDQtyWoShell.Valid || !item.IDQtyWoSize.Valid {
			return nil, false
		}
		shell, ok := shells[item.IDQtyWoShell.Int32]
		if !ok {
			return nil, false
		}
		size, ok := shell.sizes[item.IDQtyWoSize.Int32]
		if !ok {
			return nil, false
		}
		selected = append(selected, size)
	default:
		return nil, false
	}
	return selected, len(selected) > 0
}

func materialListExportOrderTotal(selected []exportShellSize) *int64 {
	if len(selected) == 0 {
		return nil
	}
	if selected[0].id == -1 {
		return int64Ptr(selected[0].order)
	}
	var total int64
	for _, size := range selected {
		total += size.order
	}
	return int64Ptr(total)
}

func materialListExportCutting(selected []exportShellSize, plans map[int32]exportPlan, actual map[int32]int64) (*int64, model.MaterialListCuttingSource) {
	if len(selected) == 0 {
		return nil, model.MaterialListCuttingNone
	}
	if selected[0].id == -1 {
		selected = selected[1:]
	}
	if len(selected) == 0 {
		return nil, model.MaterialListCuttingNone
	}
	var actualTotal int64
	hasActual := false
	for _, size := range selected {
		if qty, ok := actual[size.id]; ok {
			hasActual = true
			actualTotal += qty
		}
	}
	if hasActual {
		return int64Ptr(actualTotal), model.MaterialListCuttingActual
	}
	var plannedTotal int64
	for _, size := range selected {
		plan, ok := plans[size.id]
		if !ok || !plan.known {
			return nil, model.MaterialListCuttingNone
		}
		plannedTotal += plan.qty
	}
	return int64Ptr(plannedTotal), model.MaterialListCuttingCutPlan
}

func materialListExportTransactionsSJ(rows []entity.ListMaterialListExportSuratJalanByDateRow) (map[int32]map[time.Time]int64, []time.Time) {
	values := map[int32]map[time.Time]int64{}
	dates := map[time.Time]struct{}{}
	for _, row := range rows {
		date := row.Tanggal.Time
		if values[row.IDMaterialListItem] == nil {
			values[row.IDMaterialListItem] = map[time.Time]int64{}
		}
		values[row.IDMaterialListItem][date] = row.Qty
		dates[date] = struct{}{}
	}
	return values, sortedDates(dates)
}

func materialListExportTransactionsReceived(rows []entity.ListMaterialListExportReceivedByDateRow) (map[int32]map[time.Time]int64, []time.Time) {
	values := map[int32]map[time.Time]int64{}
	dates := map[time.Time]struct{}{}
	for _, row := range rows {
		date := row.Tanggal.Time
		if values[row.IDMaterialListItem] == nil {
			values[row.IDMaterialListItem] = map[time.Time]int64{}
		}
		values[row.IDMaterialListItem][date] = row.Qty
		dates[date] = struct{}{}
	}
	return values, sortedDates(dates)
}

func materialListExportReconciliation(rows []entity.ListMaterialListExportReconciliationRowsRow) map[int32]*entity.ListMaterialListExportReconciliationRowsRow {
	grouped := map[int32][]entity.ListMaterialListExportReconciliationRowsRow{}
	for _, row := range rows {
		if row.IDMaterialListItem.Valid {
			grouped[row.IDMaterialListItem.Int32] = append(grouped[row.IDMaterialListItem.Int32], row)
		}
	}
	result := map[int32]*entity.ListMaterialListExportReconciliationRowsRow{}
	for id, matches := range grouped {
		if len(matches) == 1 {
			row := matches[0]
			result[id] = &row
		}
	}
	return result
}

func sortedShellIDs(shells map[int32]exportShell) []int32 {
	ids := make([]int32, 0, len(shells))
	for id := range shells {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
func indexForSize(ids []int32, wanted int32) int {
	for i, id := range ids {
		if id == wanted {
			return i
		}
	}
	return -1
}
func int64Ptr(v int64) *int64             { return &v }
func exportFloat64Ptr(v float64) *float64 { return &v }
func sumTransactions(values map[time.Time]int64) int64 {
	var total int64
	for _, value := range values {
		total += value
	}
	return total
}
func sortedDates(values map[time.Time]struct{}) []time.Time {
	dates := make([]time.Time, 0, len(values))
	for date := range values {
		dates = append(dates, date)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	return dates
}
