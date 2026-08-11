package usecase

import (
	"context"
	"errors"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"permatatex-inventory/internal/entity"
)

func TestSpreadingCuttingPlanExportComposition(t *testing.T) {
	export, err := composeSpreadingCuttingPlanExport(spreadingCuttingPlanExportFixture())
	if err != nil {
		t.Fatal(err)
	}
	if export.Header.Buyer != "Buyer" || export.Header.Style != "Style" || export.Header.Model != "Model" {
		t.Fatalf("header = %+v", export.Header)
	}
	if export.Header.RatioPO == nil || *export.Header.RatioPO != "1-2-3-2" {
		t.Fatalf("ratio PO = %v", export.Header.RatioPO)
	}
	if export.AllowanceHeader != "1%" {
		t.Fatalf("allowance header = %q", export.AllowanceHeader)
	}
	if len(export.Sizes) != 4 || export.Sizes[3].Label != "XL" {
		t.Fatalf("dynamic sizes = %+v", export.Sizes)
	}
	if len(export.Shells) != 2 || export.Shells[0].ShellID == export.Shells[1].ShellID || export.Shells[0].Color != "NAVY" || export.Shells[1].Color != "BLACK" {
		t.Fatalf("multi-color shells = %+v", export.Shells)
	}
	if export.Shells[0].ReceivedActual == nil || *export.Shells[0].ReceivedActual != 1000 || export.Shells[1].ReceivedActual == nil || *export.Shells[1].ReceivedActual != 500 {
		t.Fatalf("received per shell = %v %v", export.Shells[0].ReceivedActual, export.Shells[1].ReceivedActual)
	}
	first := export.Shells[0].Ratios[0]
	if first.PlannedQty[0] == nil || *first.PlannedQty[0] != 3 || first.PlannedQty[1] == nil || *first.PlannedQty[1] != 0 || first.PlannedQty[3] != nil {
		t.Fatalf("rounded/zero/unavailable planned qty = %+v", first.PlannedQty)
	}
	if first.PlannedTotal != 8 || math.Abs(first.AllowanceConsumption-0.02) > 0.000001 || math.Abs(first.ConsPerPC-2.02) > 0.000001 {
		t.Fatalf("first ratio calculations = %+v", first)
	}
	if math.Abs(first.SubConsumption-16.16) > 0.000001 || first.SambunganYard != 6 || math.Abs(first.TotalConsumption-22.16) > 0.000001 {
		t.Fatalf("fabric usage = %+v", first)
	}
	if first.RunningBalance == nil || math.Abs(*first.RunningBalance-976.84) > 0.000001 {
		t.Fatalf("first running balance = %v", first.RunningBalance)
	}
	second := export.Shells[0].Ratios[1]
	if second.RunningBalance == nil || math.Abs(*second.RunningBalance-974.82) > 0.000001 || export.Shells[0].Subtotal.Balance == nil || math.Abs(*export.Shells[0].Subtotal.Balance-974.82) > 0.000001 {
		t.Fatalf("shell-local running balance = %+v %+v", second.RunningBalance, export.Shells[0].Subtotal.Balance)
	}
	if export.Shells[1].Ratios[0].RunningBalance == nil || math.Abs(*export.Shells[1].Ratios[0].RunningBalance-489.42) > 0.000001 {
		t.Fatalf("second shell balance carried from previous shell: %v", export.Shells[1].Ratios[0].RunningBalance)
	}
	wantOrder := []int64{30, 60, 90, 60}
	wantPlanned := []int64{5, 3, 7, 3}
	wantBalance := []int64{-25, -57, -83, -57}
	for index := range wantOrder {
		if export.Totals.OrderQty[index] != wantOrder[index] || export.Totals.PlannedQty[index] == nil || *export.Totals.PlannedQty[index] != wantPlanned[index] || export.Totals.BalanceQty[index] == nil || *export.Totals.BalanceQty[index] != wantBalance[index] {
			t.Fatalf("totals[%d] = order %d planned %v balance %v", index, export.Totals.OrderQty[index], export.Totals.PlannedQty[index], export.Totals.BalanceQty[index])
		}
	}
	if export.Totals.OrderTotal != 240 || export.Totals.PlannedTotal != 18 || export.Totals.BalanceTotal != -222 {
		t.Fatalf("document totals = %+v", export.Totals)
	}
}

func TestSpreadingCuttingPlanExportMixedAllowanceUsesNeutralHeader(t *testing.T) {
	fixture := spreadingCuttingPlanExportFixture()
	for index := range fixture.ratios {
		if fixture.ratios[index].IDRatioSpreading == 1002 {
			fixture.ratios[index].Allowance = spreadingCuttingPlanNumeric(2)
		}
	}
	export, err := composeSpreadingCuttingPlanExport(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if export.AllowanceHeader != "ALLOW." {
		t.Fatalf("mixed allowance header = %q", export.AllowanceHeader)
	}
	if got := export.Shells[0].Ratios[1].AllowanceConsumption; math.Abs(got-0.02) > 0.000001 {
		t.Fatalf("mixed row allowance amount = %v", got)
	}
}

func TestSpreadingCuttingPlanExportDistinctShellsWithEqualLabelsAreNotMerged(t *testing.T) {
	fixture := spreadingCuttingPlanExportFixture()
	for index := range fixture.shellSizes {
		if fixture.shellSizes[index].IDWoShell == 11 {
			fixture.shellSizes[index].Color = "NAVY"
		}
	}
	export, err := composeSpreadingCuttingPlanExport(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if len(export.Shells) != 2 || export.Shells[0].ShellID == export.Shells[1].ShellID || export.Shells[0].Color != "NAVY" || export.Shells[1].Color != "NAVY" {
		t.Fatalf("equal labels merged exact shell identities: %+v", export.Shells)
	}
}

func TestSpreadingCuttingPlanExportConflicts(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*spreadingCuttingPlanExportSnapshot)
	}{
		{
			name: "foreign work order ratio shell",
			mutate: func(fixture *spreadingCuttingPlanExportSnapshot) {
				fixture.ratios[0].RatioShellWoID = 999
			},
		},
		{
			name: "ratio size shell mismatch",
			mutate: func(fixture *spreadingCuttingPlanExportSnapshot) {
				fixture.ratios[0].RatioSizeShellID = pgtype.Int4{Int32: 11, Valid: true}
			},
		},
		{
			name: "duplicate exact ratio size",
			mutate: func(fixture *spreadingCuttingPlanExportSnapshot) {
				duplicate := fixture.ratios[0]
				duplicate.IDRatioSizeSpreading = pgtype.Int4{Int32: 9999, Valid: true}
				fixture.ratios = append(fixture.ratios, duplicate)
			},
		},
		{
			name: "different normalized ratio PO",
			mutate: func(fixture *spreadingCuttingPlanExportSnapshot) {
				for index := range fixture.shellSizes {
					if fixture.shellSizes[index].IDWoShell == 11 && fixture.shellSizes[index].IDSize.Int32 == 3 {
						fixture.shellSizes[index].OrderQty = pgtype.Int4{Int32: 40, Valid: true}
					}
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := spreadingCuttingPlanExportFixture()
			test.mutate(&fixture)
			_, err := composeSpreadingCuttingPlanExport(fixture)
			if !errors.Is(err, ErrSpreadingCuttingPlanExportDataConflict) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestSpreadingCuttingPlanExportAllZeroRatioPOIsBlank(t *testing.T) {
	fixture := spreadingCuttingPlanExportFixture()
	for index := range fixture.shellSizes {
		fixture.shellSizes[index].OrderQty = pgtype.Int4{Int32: 0, Valid: true}
	}
	export, err := composeSpreadingCuttingPlanExport(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if export.Header.RatioPO != nil {
		t.Fatalf("all-zero ratio PO = %v, want blank", export.Header.RatioPO)
	}
}

type spreadingCuttingPlanExportRepoStub struct {
	spreadingCuttingPlanExportSnapshot
	calls int
}

func (stub *spreadingCuttingPlanExportRepoStub) GetSpreadingCuttingPlanExportHeader(context.Context, int32) (entity.GetSpreadingCuttingPlanExportHeaderRow, error) {
	stub.calls++
	return stub.header, nil
}

func (stub *spreadingCuttingPlanExportRepoStub) ListSpreadingCuttingPlanExportShellSizes(context.Context, int32) ([]entity.ListSpreadingCuttingPlanExportShellSizesRow, error) {
	stub.calls++
	return stub.shellSizes, nil
}

func (stub *spreadingCuttingPlanExportRepoStub) ListSpreadingCuttingPlanExportRatios(context.Context, int32) ([]entity.ListSpreadingCuttingPlanExportRatiosRow, error) {
	stub.calls++
	return stub.ratios, nil
}

func (stub *spreadingCuttingPlanExportRepoStub) ListSpreadingCuttingPlanExportReceivedByShell(context.Context, int32) ([]entity.ListSpreadingCuttingPlanExportReceivedByShellRow, error) {
	stub.calls++
	return stub.received, nil
}

func TestSpreadingCuttingPlanExportUsesOnlyFixedNonMarkerReads(t *testing.T) {
	stub := &spreadingCuttingPlanExportRepoStub{spreadingCuttingPlanExportSnapshot: spreadingCuttingPlanExportFixture()}
	useCase, err := NewSpreadingCuttingPlanExportUseCase(stub)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.Compose(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if stub.calls != 4 {
		t.Fatalf("repository calls = %d, want four fixed reads without Marker Plan", stub.calls)
	}
}

func spreadingCuttingPlanExportFixture() spreadingCuttingPlanExportSnapshot {
	date := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	fixture := spreadingCuttingPlanExportSnapshot{
		header: entity.GetSpreadingCuttingPlanExportHeaderRow{
			IDSpreadingCuttingPlan: 1, NoDokumen: "SCP-001", TanggalEfektif: pgtype.Date{Time: date, Valid: true},
			IDWo: 100, Buyer: "Buyer", Model: "Model", Style: "Style",
		},
		shellSizes: []entity.ListSpreadingCuttingPlanExportShellSizesRow{
			spreadingCuttingPlanShellSize(10, "NAVY", 101, 1, "S", 10),
			spreadingCuttingPlanShellSize(10, "NAVY", 102, 2, "M", 20),
			spreadingCuttingPlanShellSize(10, "NAVY", 103, 3, "L", 30),
			spreadingCuttingPlanShellSize(10, "NAVY", 104, 4, "XL", 20),
			spreadingCuttingPlanShellSize(11, "BLACK", 201, 1, "S", 20),
			spreadingCuttingPlanShellSize(11, "BLACK", 202, 2, "M", 40),
			spreadingCuttingPlanShellSize(11, "BLACK", 203, 3, "L", 60),
			spreadingCuttingPlanShellSize(11, "BLACK", 204, 4, "XL", 40),
		},
		received: []entity.ListSpreadingCuttingPlanExportReceivedByShellRow{
			{IDWoShell: pgtype.Int4{Int32: 10, Valid: true}, SourceShellWoID: 100, ReceivedActual: 1000},
			{IDWoShell: pgtype.Int4{Int32: 11, Valid: true}, SourceShellWoID: 100, ReceivedActual: 500},
		},
	}
	fixture.ratios = append(fixture.ratios,
		spreadingCuttingPlanRatio(1, 1001, 10, 101, 1, 1, 2, 2.5, 1, 2, 3, 1),
		spreadingCuttingPlanRatio(1, 1001, 10, 102, 2, 0, 2, 2.5, 1, 2, 3, 1),
		spreadingCuttingPlanRatio(1, 1001, 10, 103, 3, 2, 2, 2.5, 1, 2, 3, 1),
		spreadingCuttingPlanRatio(2, 1002, 10, 101, 1, 0, 1, 1, 1, 0, 0, 0),
		spreadingCuttingPlanRatio(2, 1002, 10, 102, 2, 1, 1, 1, 1, 0, 0, 0),
		spreadingCuttingPlanRatio(2, 1002, 10, 103, 3, 0, 1, 1, 1, 0, 0, 0),
		spreadingCuttingPlanRatio(2, 1002, 10, 104, 4, 1, 1, 1, 1, 0, 0, 0),
		spreadingCuttingPlanRatio(3, 2001, 11, 201, 1, 1, 1, 2, 1, 1, 2, 0.5),
		spreadingCuttingPlanRatio(3, 2001, 11, 202, 2, 1, 1, 2, 1, 1, 2, 0.5),
		spreadingCuttingPlanRatio(3, 2001, 11, 203, 3, 1, 1, 2, 1, 1, 2, 0.5),
		spreadingCuttingPlanRatio(3, 2001, 11, 204, 4, 1, 1, 2, 1, 1, 2, 0.5),
	)
	return fixture
}

func spreadingCuttingPlanShellSize(shellID int32, color string, shellSizeID, sizeID int32, label string, qty int32) entity.ListSpreadingCuttingPlanExportShellSizesRow {
	return entity.ListSpreadingCuttingPlanExportShellSizesRow{
		IDWoShell: shellID, Color: color,
		IDWoShellSize: pgtype.Int4{Int32: shellSizeID, Valid: true},
		IDSize:        pgtype.Int4{Int32: sizeID, Valid: true}, NamaSize: pgtype.Text{String: label, Valid: true},
		OrderQty: pgtype.Int4{Int32: qty, Valid: true},
	}
}

func spreadingCuttingPlanRatio(componentID, ratioID, shellID, shellSizeID, sizeID, ratioPlan int32, cons, spread, allowance float64, roll, joined int32, reject float64) entity.ListSpreadingCuttingPlanExportRatiosRow {
	return entity.ListSpreadingCuttingPlanExportRatiosRow{
		IDKomponenSpreading: componentID, NamaKomponen: "Body", IDRatioSpreading: ratioID,
		RatioShellID: shellID, RatioShellWoID: 100,
		Cons: spreadingCuttingPlanNumeric(cons), PlanSpreadingGelaran: spreadingCuttingPlanNumeric(spread),
		Allowance: spreadingCuttingPlanNumeric(allowance), RollQty: roll, SambunganRoll: joined,
		Reject: spreadingCuttingPlanNumeric(reject), LebarKain: spreadingCuttingPlanNumeric(58), Ket: "note",
		IDRatioSizeSpreading: pgtype.Int4{Int32: shellSizeID + ratioID, Valid: true},
		IDWoShellSize:        pgtype.Int4{Int32: shellSizeID, Valid: true}, RatioPlan: pgtype.Int4{Int32: ratioPlan, Valid: true},
		RatioSizeShellID: pgtype.Int4{Int32: shellID, Valid: true}, RatioSizeShellWoID: pgtype.Int4{Int32: 100, Valid: true},
		IDSize: pgtype.Int4{Int32: sizeID, Valid: true},
	}
}

func spreadingCuttingPlanNumeric(value float64) pgtype.Numeric {
	var numeric pgtype.Numeric
	if err := numeric.Scan(strconv.FormatFloat(value, 'f', -1, 64)); err != nil {
		panic(err)
	}
	return numeric
}
