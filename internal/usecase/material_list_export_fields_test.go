package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

type materialListApplicabilityRepoStub struct {
	shell, size, shellSize, sourceShell, sourceTrim bool
	err                                             error
}

func (s materialListApplicabilityRepoStub) MaterialListHasApplicabilityShell(context.Context, entity.MaterialListHasApplicabilityShellParams) (bool, error) {
	return s.shell, s.err
}

func (s materialListApplicabilityRepoStub) MaterialListHasApplicabilitySize(context.Context, entity.MaterialListHasApplicabilitySizeParams) (bool, error) {
	return s.size, s.err
}

func (s materialListApplicabilityRepoStub) MaterialListHasApplicabilityShellSize(context.Context, entity.MaterialListHasApplicabilityShellSizeParams) (bool, error) {
	return s.shellSize, s.err
}

func (s materialListApplicabilityRepoStub) MaterialListHasSourceShell(context.Context, entity.MaterialListHasSourceShellParams) (bool, error) {
	return s.sourceShell, s.err
}

func (s materialListApplicabilityRepoStub) MaterialListHasSourceTrim(context.Context, entity.MaterialListHasSourceTrimParams) (bool, error) {
	return s.sourceTrim, s.err
}

type materialListConsumptionRepoStub struct {
	cons  pgtype.Numeric
	err   error
	calls int
	arg   entity.GetMaterialListItemConsumptionPrefillParams
}

func (s *materialListConsumptionRepoStub) GetMaterialListItemConsumptionPrefill(_ context.Context, arg entity.GetMaterialListItemConsumptionPrefillParams) (pgtype.Numeric, error) {
	s.calls++
	s.arg = arg
	return s.cons, s.err
}

func TestMaterialListApplicabilityValidation(t *testing.T) {
	ctx := context.Background()
	shellID, sizeID := int32(11), int32(7)
	repo := materialListApplicabilityRepoStub{shell: true, size: true, shellSize: true}

	for _, category := range []*string{nil, stringPtr("FABRIC"), stringPtr("SEWING"), stringPtr("PACKING")} {
		if err := validateMaterialListItemApplicability(ctx, repo, 1, materialListItemApplicability{Category: category}); err != nil {
			t.Fatalf("category %v rejected: %v", category, err)
		}
	}
	if err := validateMaterialListItemApplicability(ctx, repo, 1, materialListItemApplicability{Category: stringPtr("OTHER")}); !errors.Is(err, ErrMaterialListValidation) {
		t.Fatalf("invalid category error = %v", err)
	}

	valid := []materialListItemApplicability{
		{},
		{QtyWoScope: stringPtr(qtyWoScopeWholeWO)},
		{QtyWoScope: stringPtr(qtyWoScopeSize), IDQtyWoSize: &sizeID},
		{QtyWoScope: stringPtr(qtyWoScopeColor), IDQtyWoShell: &shellID},
		{QtyWoScope: stringPtr(qtyWoScopeColorSize), IDQtyWoShell: &shellID, IDQtyWoSize: &sizeID},
	}
	for _, fields := range valid {
		if err := validateMaterialListItemApplicability(ctx, repo, 1, fields); err != nil {
			t.Fatalf("valid applicability %+v rejected: %v", fields, err)
		}
	}

	invalid := []materialListItemApplicability{
		{IDQtyWoShell: &shellID},
		{QtyWoScope: stringPtr(qtyWoScopeWholeWO), IDQtyWoSize: &sizeID},
		{QtyWoScope: stringPtr(qtyWoScopeSize)},
		{QtyWoScope: stringPtr(qtyWoScopeSize), IDQtyWoShell: &shellID, IDQtyWoSize: &sizeID},
		{QtyWoScope: stringPtr(qtyWoScopeColor)},
		{QtyWoScope: stringPtr(qtyWoScopeColor), IDQtyWoShell: &shellID, IDQtyWoSize: &sizeID},
		{QtyWoScope: stringPtr(qtyWoScopeColorSize), IDQtyWoShell: &shellID},
		{QtyWoScope: stringPtr("INVALID")},
	}
	for _, fields := range invalid {
		if err := validateMaterialListItemApplicability(ctx, repo, 1, fields); !errors.Is(err, ErrMaterialListValidation) {
			t.Fatalf("invalid applicability %+v error = %v", fields, err)
		}
	}
}

func TestMaterialListApplicabilityOwnership(t *testing.T) {
	ctx := context.Background()
	shellID, sizeID := int32(11), int32(7)
	cases := []struct {
		name   string
		repo   materialListApplicabilityRepoStub
		fields materialListItemApplicability
	}{
		{"foreign shell", materialListApplicabilityRepoStub{}, materialListItemApplicability{QtyWoScope: stringPtr(qtyWoScopeColor), IDQtyWoShell: &shellID}},
		{"size absent from work order", materialListApplicabilityRepoStub{}, materialListItemApplicability{QtyWoScope: stringPtr(qtyWoScopeSize), IDQtyWoSize: &sizeID}},
		{"size absent from applicability shell", materialListApplicabilityRepoStub{}, materialListItemApplicability{QtyWoScope: stringPtr(qtyWoScopeColorSize), IDQtyWoShell: &shellID, IDQtyWoSize: &sizeID}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateMaterialListItemApplicability(ctx, tc.repo, 1, tc.fields); !errors.Is(err, ErrMaterialListValidation) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestMaterialListSourceOwnership(t *testing.T) {
	ctx := context.Background()
	shellID, trimID := int32(11), int32(7)
	if err := validateMaterialListItemSources(ctx, materialListApplicabilityRepoStub{sourceShell: true, sourceTrim: true}, 1, pgtype.Int4{Int32: shellID, Valid: true}, pgtype.Int4{Int32: trimID, Valid: true}); err != nil {
		t.Fatalf("same-work-order source links rejected: %v", err)
	}
	for _, tc := range []struct {
		name  string
		repo  materialListApplicabilityRepoStub
		shell pgtype.Int4
		trim  pgtype.Int4
	}{
		{"foreign source shell", materialListApplicabilityRepoStub{sourceTrim: true}, pgtype.Int4{Int32: shellID, Valid: true}, pgtype.Int4{}},
		{"foreign source trim", materialListApplicabilityRepoStub{sourceShell: true}, pgtype.Int4{}, pgtype.Int4{Int32: trimID, Valid: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateMaterialListItemSources(ctx, tc.repo, 1, tc.shell, tc.trim); !errors.Is(err, ErrMaterialListValidation) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestMaterialListConsPerPCPrefill(t *testing.T) {
	ctx := context.Background()
	trimCons, shellCons := numericForTest(t, "1.250"), numericForTest(t, "0.875")
	trimID, shellID := int32(21), int32(31)

	t.Run("explicit values including zero win", func(t *testing.T) {
		for _, explicit := range []*float64{float64Ptr(1.375), float64Ptr(0)} {
			repo := &materialListConsumptionRepoStub{cons: trimCons}
			got, err := materialListConsPerPCForCreate(ctx, repo, explicit, pgtype.Int4{Int32: shellID, Valid: true}, pgtype.Int4{Int32: trimID, Valid: true})
			if err != nil || numericValueForTest(t, got) != *explicit {
				t.Fatalf("explicit %v got %v err %v", *explicit, got, err)
			}
			if repo.calls != 0 {
				t.Fatal("explicit value must not call source prefill")
			}
		}
	})

	t.Run("trim source prefill wins over shell source", func(t *testing.T) {
		repo := &materialListConsumptionRepoStub{cons: trimCons}
		got, err := materialListConsPerPCForCreate(ctx, repo, nil, pgtype.Int4{Int32: shellID, Valid: true}, pgtype.Int4{Int32: trimID, Valid: true})
		if err != nil || numericValueForTest(t, got) != numericValueForTest(t, trimCons) || !repo.arg.IDWoTrim.Valid {
			t.Fatalf("got %v arg %+v err %v", got, repo.arg, err)
		}
	})

	t.Run("shell source prefill", func(t *testing.T) {
		repo := &materialListConsumptionRepoStub{cons: shellCons}
		got, err := materialListConsPerPCForCreate(ctx, repo, nil, pgtype.Int4{Int32: shellID, Valid: true}, pgtype.Int4{})
		if err != nil || numericValueForTest(t, got) != numericValueForTest(t, shellCons) {
			t.Fatalf("got %v err %v", got, err)
		}
	})

	t.Run("no source stays null and applicability never pre-fills", func(t *testing.T) {
		repo := &materialListConsumptionRepoStub{cons: shellCons}
		got, err := materialListConsPerPCForCreate(ctx, repo, nil, pgtype.Int4{}, pgtype.Int4{})
		if err != nil || got.Valid || repo.calls != 0 {
			t.Fatalf("got %v calls %d err %v", got, repo.calls, err)
		}
	})

	if _, err := materialListConsPerPCForCreate(ctx, &materialListConsumptionRepoStub{}, float64Ptr(-1), pgtype.Int4{}, pgtype.Int4{}); !errors.Is(err, ErrMaterialListValidation) {
		t.Fatalf("negative consumption error = %v", err)
	}
}

func TestMaterialListSourceAndApplicabilityAreIndependent(t *testing.T) {
	ctx := context.Background()
	sourceShellA, applicabilityShellB, trimID := int32(10), int32(20), int32(30)
	repo := materialListApplicabilityRepoStub{shell: true}
	if err := validateMaterialListItemApplicability(ctx, repo, 1, materialListItemApplicability{
		QtyWoScope:   stringPtr(qtyWoScopeColor),
		IDQtyWoShell: &applicabilityShellB,
	}); err != nil {
		t.Fatalf("trim source + color applicability rejected: %v", err)
	}
	trimPrefill := &materialListConsumptionRepoStub{cons: numericForTest(t, "1.125")}
	if _, err := materialListConsPerPCForCreate(ctx, trimPrefill, nil, pgtype.Int4{}, pgtype.Int4{Int32: trimID, Valid: true}); err != nil || !trimPrefill.arg.IDWoTrim.Valid {
		t.Fatalf("trim source prefill err %v arg %+v", err, trimPrefill.arg)
	}

	if err := validateMaterialListItemApplicability(ctx, repo, 1, materialListItemApplicability{
		QtyWoScope:   stringPtr(qtyWoScopeColor),
		IDQtyWoShell: &applicabilityShellB,
	}); err != nil {
		t.Fatalf("shell source A + color applicability shell B rejected: %v", err)
	}
	shellPrefill := &materialListConsumptionRepoStub{cons: numericForTest(t, "0.750")}
	if _, err := materialListConsPerPCForCreate(ctx, shellPrefill, nil, pgtype.Int4{Int32: sourceShellA, Valid: true}, pgtype.Int4{}); err != nil || shellPrefill.arg.IDWoShell.Int32 != sourceShellA {
		t.Fatalf("shell source prefill err %v arg %+v", err, shellPrefill.arg)
	}
}

func TestMaterialListUpdateNeverPrefillsConsumption(t *testing.T) {
	source, err := os.ReadFile("material_list_usecase.go")
	if err != nil {
		t.Fatal(err)
	}
	update := string(source)
	start := strings.Index(update, "func (u *MaterialListUseCase) UpdateItem")
	end := strings.Index(update[start:], "func (u *MaterialListUseCase) DeleteItem")
	if start < 0 || end < 0 {
		t.Fatal("could not locate UpdateItem source")
	}
	update = update[start : start+end]
	if strings.Contains(update, "materialListConsPerPCForCreate") || strings.Contains(update, "GetMaterialListItemConsumptionPrefill") {
		t.Fatal("PATCH must not re-prefill consumption from a source")
	}
	if !strings.Contains(update, "SetConsPerPc:       req.ConsPerPC.Set") {
		t.Fatal("PATCH must update consumption only when it was explicitly supplied")
	}
}

func TestMaterialListResponsePreservesNullableExportFields(t *testing.T) {
	createdAt := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	legacy := materialListItemResponse(1, "item", "", 0, "pcs", numericForTest(t, "0"), pgtype.Int4{}, pgtype.Int4{}, pgtype.Text{}, pgtype.Numeric{}, pgtype.Text{}, pgtype.Int4{}, pgtype.Int4{}, createdAt, 0, 0)
	if legacy.Category != nil || legacy.ConsPerPC != nil || legacy.QtyWoScope != nil || legacy.IDQtyWoShell != nil || legacy.IDQtyWoSize != nil {
		t.Fatalf("legacy nullable fields changed: %+v", legacy)
	}

	category, scope, cons, shellID, sizeID := pgtype.Text{String: "FABRIC", Valid: true}, pgtype.Text{String: qtyWoScopeColorSize, Valid: true}, numericForTest(t, "0.000"), pgtype.Int4{Int32: 10, Valid: true}, pgtype.Int4{Int32: 4, Valid: true}
	item := materialListItemResponse(1, "item", "", 0, "pcs", numericForTest(t, "0"), pgtype.Int4{Int32: 1, Valid: true}, pgtype.Int4{Int32: 2, Valid: true}, category, cons, scope, shellID, sizeID, createdAt, 0, 0)
	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"\"category\":\"FABRIC\"", "\"cons_per_pc\":0", "\"qty_wo_scope\":\"COLOR_SIZE\"", "\"id_qty_wo_shell\":10", "\"id_qty_wo_size\":4"} {
		if !strings.Contains(string(encoded), fragment) {
			t.Fatalf("response missing %s: %s", fragment, encoded)
		}
	}
}

func TestOptionalPatchValues(t *testing.T) {
	var omitted, explicitNull, explicitZero struct {
		ConsPerPC model.OptionalFloat64 `json:"cons_per_pc"`
	}
	if err := json.Unmarshal([]byte(`{}`), &omitted); err != nil || omitted.ConsPerPC.Set {
		t.Fatalf("omitted = %+v err %v", omitted.ConsPerPC, err)
	}
	if err := json.Unmarshal([]byte(`{"cons_per_pc":null}`), &explicitNull); err != nil || !explicitNull.ConsPerPC.Set || explicitNull.ConsPerPC.Value != nil {
		t.Fatalf("null = %+v err %v", explicitNull.ConsPerPC, err)
	}
	if err := json.Unmarshal([]byte(`{"cons_per_pc":0}`), &explicitZero); err != nil || !explicitZero.ConsPerPC.Set || explicitZero.ConsPerPC.Value == nil || *explicitZero.ConsPerPC.Value != 0 {
		t.Fatalf("zero = %+v err %v", explicitZero.ConsPerPC, err)
	}
}

func TestOldStyleMaterialListPayloadRemainsCompatible(t *testing.T) {
	var request model.CreateMaterialListItemBody
	if err := json.Unmarshal([]byte(`{"item":"Legacy item","unit":"pcs","qty":10,"est_price":1.5}`), &request); err != nil {
		t.Fatal(err)
	}
	if request.Category != nil || request.ConsPerPC != nil || request.QtyWoScope != nil || request.IDQtyWoShell != nil || request.IDQtyWoSize != nil {
		t.Fatalf("old-style request unexpectedly set Stage 1 fields: %+v", request)
	}
}

func TestWorkOrderCreateUsesMaterialListExportFields(t *testing.T) {
	source, err := os.ReadFile("work_order_production_usecase.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"if idx < 0 || idx >= len(recordedShells)",
		"if idx < 0 || idx >= len(recordedTrims)",
		"QtyWoShellIndex",
		"validateMaterialListItemApplicability(ctx, qtx",
		"materialListConsPerPCForCreate(ctx, qtx",
		"IDQtyWoShell:   idQtyWoShell",
		"IDQtyWoSize:    nullableInt32Param(itemReq.IDQtyWoSize)",
		"mli.Category, mli.ConsPerPc, mli.QtyWoScope, mli.IDQtyWoShell, mli.IDQtyWoSize",
	} {
		if !strings.Contains(string(source), fragment) {
			t.Fatalf("work order material-list creation missing %q", fragment)
		}
	}
}

func numericForTest(t *testing.T, value string) pgtype.Numeric {
	t.Helper()
	var numeric pgtype.Numeric
	if err := numeric.Scan(value); err != nil {
		t.Fatal(err)
	}
	return numeric
}

func numericValueForTest(t *testing.T, value pgtype.Numeric) float64 {
	t.Helper()
	parsed, err := value.Float64Value()
	if err != nil || !parsed.Valid {
		t.Fatalf("numeric value %v: %v", value, err)
	}
	return parsed.Float64
}

func stringPtr(value string) *string    { return &value }
func float64Ptr(value float64) *float64 { return &value }
