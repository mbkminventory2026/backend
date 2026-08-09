package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

func TestMaterialListExportCompositionPreservesBusinessData(t *testing.T) {
	date1 := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	snapshot := materialListExportSnapshot{
		header: entity.GetMaterialListExportHeaderRow{IDMaterialList: 1, MaterialListName: "ML-A", IDWo: 2, Buyer: "Buyer", Model: "Model", Style: "Style", WoQty: 99, FobCmt: true, Delivery: pgtype.Date{Time: date2, Valid: true}},
		shellSizes: []entity.ListMaterialListExportShellSizesRow{
			shellSize(10, "Navy", 101, 1, "S", 10), shellSize(10, "Navy", 102, 2, "M", 20), shellSize(11, "Navy", 103, 2, "M", 30),
		},
		plans: []entity.ListMaterialListExportMarkerPlansRow{{IDMarkerPlan: 20, IDWoShell: 10}, {IDMarkerPlan: 21, IDWoShell: 11}},
		ratios: []entity.ListMaterialListExportMarkerRatiosRow{
			ratio(20, 10, 101, 1, 2, 5), ratio(20, 10, 102, 2, 4, 5), ratio(21, 11, 103, 2, 6, 5),
		},
		cutting: []entity.ListMaterialListExportCuttingRow{{IDWoShellSize: 101, ActualCutQty: 0}, {IDWoShellSize: 102, ActualCutQty: 4}},
		items: []entity.ListMaterialListExportItemsRow{
			item(1, "WHOLE_WO", 0, 0, "1.5", 10, 0, 7),
			item(2, "SIZE", 0, 2, "2", 0, 0, 0),
			item(3, "COLOR", 11, 0, "3", 0, 90, 8),
			item(4, "COLOR_SIZE", 10, 1, "4", 0, 0, 0),
			item(5, "", 0, 0, "9", 0, 0, 0),
		},
		sj:             []entity.ListMaterialListExportSuratJalanByDateRow{{IDMaterialListItem: 1, Tanggal: pgtype.Date{Time: date1, Valid: true}, Qty: 4}, {IDMaterialListItem: 1, Tanggal: pgtype.Date{Time: date2, Valid: true}, Qty: 2}},
		received:       []entity.ListMaterialListExportReceivedByDateRow{{IDMaterialListItem: 1, Tanggal: pgtype.Date{Time: date1, Valid: true}, Qty: 10}},
		reconciliation: []entity.ListMaterialListExportReconciliationRowsRow{{IDMaterialListItem: pgtype.Int4{Int32: 1, Valid: true}, RejectQty: 1, ReturQty: 2, Keterangan: "row note"}},
	}
	got, err := composeMaterialListExport(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if got.Header.WorkOrderQty != 99 || got.Header.Style != "Style" || !got.Header.FobCmt || !got.Header.Delivery.Equal(date2) {
		t.Fatalf("header = %+v", got.Header)
	}
	if len(got.DetailQty.Shells) != 2 || got.DetailQty.Shells[0].ShellID == got.DetailQty.Shells[1].ShellID || got.DetailQty.Shells[0].Color != "Navy" || got.DetailQty.Shells[1].Color != "Navy" {
		t.Fatalf("duplicate colors merged: %+v", got.DetailQty.Shells)
	}
	if got.DetailQty.Shells[1].Values[1].Order != nil {
		t.Fatal("missing shell-size must remain blank")
	}
	if *got.DetailQty.Summaries[0].OrderTotal != 50 || *got.DetailQty.Shells[0].Values[1].ActualCut != 0 {
		t.Fatalf("detail totals/actual zero = %+v", got.DetailQty)
	}
	// Balance is a per-size workbook summary: TOTAL ACT CUT - RMS/TOTAL.
	if *got.DetailQty.Summaries[1].RMSTotal != 10 || *got.DetailQty.Summaries[1].TotalActCut != 0 || *got.DetailQty.Summaries[1].Balance != -10 {
		t.Fatalf("detail summary balance = %+v", got.DetailQty.Summaries[1])
	}
	if got.DetailQty.Summaries[0].Balance != nil {
		t.Fatalf("detail balance with unavailable ACT CUT must be blank: %+v", got.DetailQty.Summaries[0])
	}
	rows := got.MaterialRows
	if *rows[0].QtyWO != 99 || *rows[1].QtyWO != 50 || *rows[2].QtyWO != 30 || *rows[3].QtyWO != 10 || rows[4].QtyWO != nil {
		t.Fatalf("qty scopes = %+v", rows)
	}
	if rows[0].CuttingSource != model.MaterialListCuttingActual || *rows[0].CuttingQty != 4 {
		t.Fatalf("whole actual = %+v", rows[0])
	}
	if rows[2].Source.TrimID == nil || rows[2].Source.AllowancePercent == nil || *rows[2].Source.AllowancePercent != 8 {
		t.Fatalf("trim source allowance = %+v", rows[2].Source)
	}
	if *rows[0].Total != 148.5 || *rows[0].ActualConsumption != 6.42 {
		t.Fatalf("stored cons / allowance = %+v", rows[0])
	}
	if *rows[0].SJTotal != 6 || len(got.SJDates) != 2 || *rows[0].ReceivedTotal != 10 || *rows[0].BalanceReceived != 4 {
		t.Fatalf("transactions = %+v", rows[0])
	}
	if *rows[0].RejectRetur != 3 || math.Abs(*rows[0].FinalBalance-0.58) > 0.000001 || rows[1].RejectRetur != nil || rows[1].FinalBalance != nil {
		t.Fatalf("reconciliation/final = %+v %+v", rows[0], rows[1])
	}
}

func TestMaterialListExportMarkerPlanConflictsAndFallback(t *testing.T) {
	shells, _, _ := materialListExportShells([]entity.ListMaterialListExportShellSizesRow{shellSize(10, "A", 101, 1, "S", 1)})
	_, err := materialListExportPlans(shells, []entity.ListMaterialListExportMarkerPlansRow{{IDMarkerPlan: 1, IDWoShell: 10}, {IDMarkerPlan: 2, IDWoShell: 10}}, nil)
	if !errors.Is(err, ErrMaterialListExportMarkerPlanConflict) {
		t.Fatalf("multi parent error = %v", err)
	}
	_, err = materialListExportPlans(shells, []entity.ListMaterialListExportMarkerPlansRow{{IDMarkerPlan: 1, IDWoShell: 10}}, []entity.ListMaterialListExportMarkerRatiosRow{ratio(1, 11, 101, 1, 1, 1)})
	if !errors.Is(err, ErrMaterialListExportDataConflict) {
		t.Fatalf("mismatch error = %v", err)
	}
	qty, source := materialListExportCutting([]exportShellSize{{id: 101}}, map[int32]exportPlan{101: {known: true, qty: 7}}, map[int32]int64{})
	if source != model.MaterialListCuttingCutPlan || *qty != 7 {
		t.Fatalf("fallback = %v %v", source, qty)
	}
	qty, source = materialListExportCutting([]exportShellSize{{id: 101}}, map[int32]exportPlan{101: {known: true, qty: 7}}, map[int32]int64{101: 0})
	if source != model.MaterialListCuttingActual || *qty != 0 {
		t.Fatalf("actual zero = %v %v", source, qty)
	}
}

func TestMaterialListExportScopeUsesApplicabilityNotSource(t *testing.T) {
	shells, _, _ := materialListExportShells([]entity.ListMaterialListExportShellSizesRow{
		shellSize(10, "Source", 101, 1, "S", 5),
		shellSize(11, "Applicable", 102, 1, "S", 7),
	})
	row := materialListExportRow(
		item(1, qtyWoScopeColor, 11, 0, "2", 10, 0, 15),
		shells,
		nil,
		map[int32]int64{102: 3},
		nil,
		nil,
		nil,
		99,
	)
	if *row.QtyWO != 7 || *row.CuttingQty != 3 || math.Abs(*row.ActualConsumption-6.9) > 0.000001 {
		t.Fatalf("source shell must not determine scope: %+v", row)
	}
	if row.Applicability.ShellID == nil || *row.Applicability.ShellID != 11 || row.Source.ShellID == nil || *row.Source.ShellID != 10 {
		t.Fatalf("source/applicability distinction lost: %+v", row)
	}
}

func TestMaterialListExportReconciliationAmbiguityStaysUnavailable(t *testing.T) {
	rows := materialListExportReconciliation([]entity.ListMaterialListExportReconciliationRowsRow{
		{IDMaterialListItem: pgtype.Int4{Int32: 1, Valid: true}, RejectQty: 1}, {IDMaterialListItem: pgtype.Int4{Int32: 1, Valid: true}, ReturQty: 2},
	})
	if rows[1] != nil {
		t.Fatalf("ambiguous reconciliation = %+v", rows[1])
	}
}

type materialListExportRepoStub struct {
	materialListExportSnapshot
	calls int
}

func (s *materialListExportRepoStub) GetMaterialListExportHeader(context.Context, int32) (entity.GetMaterialListExportHeaderRow, error) {
	s.calls++
	return s.header, nil
}
func (s *materialListExportRepoStub) ListMaterialListExportItems(context.Context, int32) ([]entity.ListMaterialListExportItemsRow, error) {
	s.calls++
	return s.items, nil
}
func (s *materialListExportRepoStub) ListMaterialListExportShellSizes(context.Context, int32) ([]entity.ListMaterialListExportShellSizesRow, error) {
	s.calls++
	return s.shellSizes, nil
}
func (s *materialListExportRepoStub) ListMaterialListExportMarkerPlans(context.Context, int32) ([]entity.ListMaterialListExportMarkerPlansRow, error) {
	s.calls++
	return s.plans, nil
}
func (s *materialListExportRepoStub) ListMaterialListExportMarkerRatios(context.Context, int32) ([]entity.ListMaterialListExportMarkerRatiosRow, error) {
	s.calls++
	return s.ratios, nil
}
func (s *materialListExportRepoStub) ListMaterialListExportCutting(context.Context, int32) ([]entity.ListMaterialListExportCuttingRow, error) {
	s.calls++
	return s.cutting, nil
}
func (s *materialListExportRepoStub) ListMaterialListExportSuratJalanByDate(context.Context, int32) ([]entity.ListMaterialListExportSuratJalanByDateRow, error) {
	s.calls++
	return s.sj, nil
}
func (s *materialListExportRepoStub) ListMaterialListExportReceivedByDate(context.Context, int32) ([]entity.ListMaterialListExportReceivedByDateRow, error) {
	s.calls++
	return s.received, nil
}
func (s *materialListExportRepoStub) ListMaterialListExportReconciliationRows(context.Context, entity.ListMaterialListExportReconciliationRowsParams) ([]entity.ListMaterialListExportReconciliationRowsRow, error) {
	s.calls++
	return s.reconciliation, nil
}

func TestMaterialListExportUsesFixedBatchedReads(t *testing.T) {
	repo := &materialListExportRepoStub{materialListExportSnapshot: materialListExportSnapshot{header: entity.GetMaterialListExportHeaderRow{IDMaterialList: 1, IDWo: 1}}}
	uc, err := NewMaterialListExportUseCase(repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Compose(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if repo.calls != 9 {
		t.Fatalf("calls = %d, want 9 fixed batched reads", repo.calls)
	}
}

func shellSize(shell int32, color string, shellSizeID, sizeID int32, name string, qty int32) entity.ListMaterialListExportShellSizesRow {
	return entity.ListMaterialListExportShellSizesRow{IDWoShell: shell, Color: color, IDWoShellSize: pgtype.Int4{Int32: shellSizeID, Valid: true}, IDSize: pgtype.Int4{Int32: sizeID, Valid: true}, NamaSize: pgtype.Text{String: name, Valid: true}, OrderQty: pgtype.Int4{Int32: qty, Valid: true}}
}
func ratio(plan, shell, shellSizeID, sizeID, ratioPlan int32, spread float64) entity.ListMaterialListExportMarkerRatiosRow {
	return entity.ListMaterialListExportMarkerRatiosRow{IDMarkerPlan: plan, MarkerPlanShellID: shell, RatioShellID: shell, IDWoShellSize: shellSizeID, RatioPlan: ratioPlan, RatioSizeShellID: shell, IDSize: sizeID, PlanSpreadingGelaran: exportNumeric(spread)}
}
func item(id int32, scope string, shellID, sizeID int32, cons string, sourceShell, sourceTrim, allowance int32) entity.ListMaterialListExportItemsRow {
	r := entity.ListMaterialListExportItemsRow{IDMaterialListItem: id, Item: "item", Unit: "yd", ConsPerPc: exportNumericString(cons)}
	if scope != "" {
		r.QtyWoScope = pgtype.Text{String: scope, Valid: true}
	}
	if shellID != 0 {
		r.IDQtyWoShell = pgtype.Int4{Int32: shellID, Valid: true}
	}
	if sizeID != 0 {
		r.IDQtyWoSize = pgtype.Int4{Int32: sizeID, Valid: true}
	}
	if sourceShell != 0 {
		r.SourceShellID = pgtype.Int4{Int32: sourceShell, Valid: true}
		r.SourceShellAllow = pgtype.Int4{Int32: allowance, Valid: true}
	}
	if sourceTrim != 0 {
		r.SourceTrimID = pgtype.Int4{Int32: sourceTrim, Valid: true}
		r.SourceTrimAllow = pgtype.Int4{Int32: allowance, Valid: true}
	}
	return r
}
func exportNumeric(value float64) pgtype.Numeric {
	return exportNumericString(fmt.Sprintf("%g", value))
}
func exportNumericString(value string) pgtype.Numeric {
	var numeric pgtype.Numeric
	_ = numeric.Scan(value)
	return numeric
}
