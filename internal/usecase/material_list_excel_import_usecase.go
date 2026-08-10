package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

const (
	materialListImportMarker       = "MATERIAL_LIST_IMPORT_V1"
	materialListImportMaxBytes     = 2 << 20
	materialListImportMaxRows      = 1000
	materialListImportMaxRowErrors = 100
)

var decimalLiteral = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)

var materialListImportHeaders = []string{"CATEGORY", "ITEM", "DESCRIPTION", "QTY", "UOM", "EST_PRICE", "CONS_PER_PC", "SOURCE_TYPE", "SOURCE_ID", "QTY_WO_SCOPE", "APPLICABILITY_SHELL_ID", "APPLICABILITY_SIZE_ID"}

// MaterialListExcelImportError is deliberately transport-independent. Stage 6C
// can map its stable Code and safe Message to the standard HTTP envelope.
type MaterialListExcelImportError struct {
	Row     int    `json:"row,omitempty"`
	Column  string `json:"column,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type MaterialListExcelImportValidationError struct {
	Errors     []MaterialListExcelImportError
	ErrorCount int
}

func (e *MaterialListExcelImportValidationError) Error() string {
	return "material list import validation failed"
}

type MaterialListExcelImportResult struct {
	CreatedCount int
	Errors       []MaterialListExcelImportError
	ErrorCount   int
}

type MaterialListExcelImportUseCase struct {
	repo   entity.Querier
	dbPool *pgxpool.Pool
}

func NewMaterialListExcelImportUseCase(repo entity.Querier, dbPool *pgxpool.Pool) (*MaterialListExcelImportUseCase, error) {
	if repo == nil || dbPool == nil {
		return nil, errors.New("material list import repository and database pool are required")
	}
	return &MaterialListExcelImportUseCase{repo: repo, dbPool: dbPool}, nil
}

// Import accepts only workbook content, not HTTP/multipart types. It validates
// the complete workbook before beginning a write transaction.
func (u *MaterialListExcelImportUseCase) Import(ctx context.Context, targetID int32, reader io.Reader) (*MaterialListExcelImportResult, error) {
	rows, validation := parseMaterialListImportWorkbook(targetID, reader)
	if validation.ErrorCount != 0 {
		return validationResult(validation), validation
	}

	ml, err := u.repo.GetMaterialList(ctx, targetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, importDomainError("material_list_not_found", "target material list was not found")
		}
		return nil, fmt.Errorf("%w: load target material list: %v", ErrMaterialListUnavailable, err)
	}
	if ml.IsLocked {
		return nil, importDomainError("material_list_locked", "target material list is locked")
	}

	validation = validateMaterialListImportRows(ctx, u.repo, targetID, rows)
	existing, err := u.repo.ListMaterialListItemsByML(ctx, targetID)
	if err != nil {
		return nil, fmt.Errorf("%w: load material list items: %v", ErrMaterialListUnavailable, err)
	}
	validateMaterialListImportDuplicates(rows, existing, validation)
	if validation.ErrorCount != 0 {
		return validationResult(validation), validation
	}

	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: begin material list import transaction", ErrMaterialListUnavailable)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	txRepo := entity.New(tx)
	lockedML, err := txRepo.GetMaterialListForUpdate(ctx, targetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, importDomainError("material_list_not_found", "target material list was not found")
		}
		return nil, fmt.Errorf("%w: lock target material list: %v", ErrMaterialListUnavailable, err)
	}
	if lockedML.IsLocked {
		return nil, importDomainError("material_list_locked", "target material list is locked")
	}
	validation = validateMaterialListImportRows(ctx, txRepo, targetID, rows)
	existing, err = txRepo.ListMaterialListItemsByML(ctx, targetID)
	if err != nil {
		return nil, fmt.Errorf("%w: recheck material list items: %v", ErrMaterialListUnavailable, err)
	}
	validateMaterialListImportDuplicates(rows, existing, validation)
	if validation.ErrorCount != 0 {
		return validationResult(validation), validation
	}
	for _, row := range rows {
		if _, err := createMaterialListItemWithNumeric(ctx, txRepo, targetID, row.request, row.priceNumeric, row.consNumeric); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: commit material list import transaction", ErrMaterialListUnavailable)
	}
	return &MaterialListExcelImportResult{CreatedCount: len(rows)}, nil
}

type parsedMaterialListImportRow struct {
	row          int
	request      model.CreateMaterialListItemBody
	key          string
	priceText    string
	consText     string
	priceNumeric pgtype.Numeric
	consNumeric  *pgtype.Numeric
}

func parseMaterialListImportWorkbook(targetID int32, reader io.Reader) ([]parsedMaterialListImportRow, *MaterialListExcelImportValidationError) {
	validation := &MaterialListExcelImportValidationError{}
	content, err := io.ReadAll(io.LimitReader(reader, materialListImportMaxBytes+1))
	if err != nil || len(content) > materialListImportMaxBytes {
		addImportError(validation, MaterialListExcelImportError{Code: "invalid_workbook", Message: "workbook must be a valid XLSX file no larger than 2 MiB"})
		return nil, validation
	}
	book, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		addImportError(validation, MaterialListExcelImportError{Code: "invalid_workbook", Message: "workbook must be a valid XLSX file"})
		return nil, validation
	}
	defer book.Close()
	gotSheets := book.GetSheetList()
	if len(gotSheets) != 3 || gotSheets[0] != "IMPORT" || gotSheets[1] != "REFERENSI" || gotSheets[2] != "_META" {
		addImportError(validation, MaterialListExcelImportError{Code: "invalid_workbook", Message: "workbook must contain exactly IMPORT, REFERENSI, and _META sheets"})
		return nil, validation
	}
	for _, sheet := range []string{"IMPORT", "REFERENSI"} {
		visible, err := book.GetSheetVisible(sheet)
		if err != nil || !visible {
			addImportError(validation, MaterialListExcelImportError{Code: "invalid_workbook", Message: "only _META may be hidden"})
		}
	}
	for _, sheet := range []string{"IMPORT", "_META"} {
		merges, err := book.GetMergeCells(sheet)
		if err != nil || len(merges) != 0 {
			addImportError(validation, MaterialListExcelImportError{Code: "invalid_workbook", Message: "merged cells are not allowed in structural sheets"})
		}
		for rowIndex, cells := range mustRows(book, sheet) {
			for colIndex := range cells {
				cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+1)
				formula, _ := book.GetCellFormula(sheet, cell)
				if formula != "" {
					addImportError(validation, MaterialListExcelImportError{Row: rowIndex + 1, Column: cell, Code: "formula_not_allowed", Message: "formulas are not allowed in structural sheets"})
				}
			}
		}
	}
	marker, _ := book.GetCellValue("_META", "A1")
	if marker != materialListImportMarker {
		addImportError(validation, MaterialListExcelImportError{Column: "_META!A1", Code: "unsupported_template_version", Message: "unsupported material list import template version"})
	}
	metaTarget, invalidMetaTarget := parsePositiveID(strings.TrimSpace(value(book, "_META", "B1")))
	if invalidMetaTarget || metaTarget != targetID {
		addImportError(validation, MaterialListExcelImportError{Column: "_META!B1", Code: "target_material_list_mismatch", Message: "workbook target material list does not match import target"})
	}
	rows := mustRows(book, "IMPORT")
	if len(rows) == 0 {
		addImportError(validation, MaterialListExcelImportError{Code: "invalid_workbook", Message: "IMPORT headers are required"})
		return nil, validation
	}
	for i, header := range materialListImportHeaders {
		actual := ""
		if i < len(rows[0]) {
			actual = rows[0][i]
		}
		if actual != header {
			addImportError(validation, MaterialListExcelImportError{Row: 1, Column: header, Code: "invalid_workbook", Message: "IMPORT headers must match the required order"})
		}
	}
	if len(rows[0]) > len(materialListImportHeaders) && rowHasText(rows[0]) {
		addImportError(validation, MaterialListExcelImportError{Row: 1, Code: "invalid_workbook", Message: "IMPORT has unexpected columns"})
	}
	if validation.ErrorCount != 0 {
		return nil, validation
	}

	var parsed []parsedMaterialListImportRow
	logicalRows := 0
	gap := false
	for i := 1; i < len(rows); i++ {
		values := fixedRow(rows[i], len(materialListImportHeaders))
		populated := rowHasText(rows[i])
		if !populated {
			if len(parsed) > 0 {
				gap = true
			}
			continue
		}
		logicalRows++
		if logicalRows > materialListImportMaxRows {
			addImportError(validation, MaterialListExcelImportError{Row: i + 1, Code: "row_limit_exceeded", Message: "workbook may contain at most 1000 data rows"})
			continue
		}
		if gap {
			addImportError(validation, MaterialListExcelImportError{Row: i + 1, Code: "malformed_row", Message: "populated rows may not follow a blank row"})
			continue
		}
		if len(rows[i]) > len(materialListImportHeaders) {
			addImportError(validation, MaterialListExcelImportError{Row: i + 1, Code: "malformed_row", Message: "row has unexpected populated columns"})
			continue
		}
		parsed = append(parsed, parseMaterialListImportRow(i+1, values, validation))
	}
	return parsed, validation
}

func parseMaterialListImportRow(row int, v []string, validation *MaterialListExcelImportValidationError) parsedMaterialListImportRow {
	text := func(i int) string { return strings.TrimSpace(v[i]) }
	category := strings.ToUpper(text(0))
	item, description, unit := text(1), text(2), text(4)
	if category != "FABRIC" && category != "SEWING" && category != "PACKING" {
		addFieldError(validation, row, "CATEGORY", "invalid_enum", "CATEGORY must be FABRIC, SEWING, or PACKING")
	}
	if item == "" {
		addFieldError(validation, row, "ITEM", "required", "ITEM is required")
	} else if len(item) > 100 {
		addFieldError(validation, row, "ITEM", "invalid_reference", "ITEM exceeds maximum length")
	}
	if unit == "" {
		addFieldError(validation, row, "UOM", "required", "UOM is required")
	} else if len(unit) > 20 {
		addFieldError(validation, row, "UOM", "invalid_reference", "UOM exceeds maximum length")
	}
	qty, ok := parseQty(text(3))
	if !ok {
		addFieldError(validation, row, "QTY", "invalid_integer", "QTY must be a non-negative integer")
	}
	price, priceText, err := parseDecimal(text(5), 15, 2)
	priceNumeric := numericFromDecimal(priceText)
	if err != "" {
		addFieldError(validation, row, "EST_PRICE", err, "EST_PRICE must be a non-negative decimal with at most 2 decimal places")
	}
	consRaw := text(6)
	var cons *float64
	consText := ""
	var consNumeric *pgtype.Numeric
	if consRaw != "" {
		var n float64
		n, consText, err = parseDecimal(consRaw, 15, 3)
		if err != "" {
			addFieldError(validation, row, "CONS_PER_PC", err, "CONS_PER_PC must be a non-negative decimal with at most 3 decimal places")
		} else {
			cons = &n
			numeric := numericFromDecimal(consText)
			consNumeric = &numeric
		}
	}
	sourceType := strings.ToUpper(text(7))
	sourceIDRaw := text(8)
	var shell, trim *int32
	sourceID, sourceErr := parsePositiveID(sourceIDRaw)
	switch sourceType {
	case "NONE":
		if sourceIDRaw != "" {
			sourceErr = true
		}
	case "SHELL":
		if sourceErr {
			addFieldError(validation, row, "SOURCE_ID", "invalid_source", "SHELL source requires a positive source ID")
		} else {
			shell = &sourceID
		}
	case "TRIM":
		if sourceErr {
			addFieldError(validation, row, "SOURCE_ID", "invalid_source", "TRIM source requires a positive source ID")
		} else {
			trim = &sourceID
		}
	default:
		addFieldError(validation, row, "SOURCE_TYPE", "invalid_enum", "SOURCE_TYPE must be NONE, SHELL, or TRIM")
	}
	if sourceType == "NONE" && sourceIDRaw != "" {
		addFieldError(validation, row, "SOURCE_ID", "invalid_source", "NONE source must not have a source ID")
	}
	scope := strings.ToUpper(text(9))
	shellID, shellErr := parsePositiveID(text(10))
	sizeID, sizeErr := parsePositiveID(text(11))
	var appShell, appSize *int32
	switch scope {
	case qtyWoScopeWholeWO:
		if text(10) != "" || text(11) != "" {
			addFieldError(validation, row, "APPLICABILITY_SHELL_ID", "invalid_applicability", "WHOLE_WO requires blank applicability IDs")
		}
	case qtyWoScopeSize:
		if text(10) != "" || sizeErr {
			addFieldError(validation, row, "APPLICABILITY_SIZE_ID", "invalid_applicability", "SIZE requires a positive size ID and blank shell ID")
		} else {
			appSize = &sizeID
		}
	case qtyWoScopeColor:
		if shellErr || text(11) != "" {
			addFieldError(validation, row, "APPLICABILITY_SHELL_ID", "invalid_applicability", "COLOR requires a positive shell ID and blank size ID")
		} else {
			appShell = &shellID
		}
	case qtyWoScopeColorSize:
		if shellErr || sizeErr {
			addFieldError(validation, row, "APPLICABILITY_SHELL_ID", "invalid_applicability", "COLOR_SIZE requires positive shell and size IDs")
		} else {
			appShell = &shellID
			appSize = &sizeID
		}
	default:
		addFieldError(validation, row, "QTY_WO_SCOPE", "invalid_enum", "QTY_WO_SCOPE must be WHOLE_WO, SIZE, COLOR, or COLOR_SIZE")
	}
	request := model.CreateMaterialListItemBody{Item: item, Description: description, Qty: qty, Unit: unit, EstPrice: price, IDWoShell: shell, IDWoTrim: trim, Category: &category, ConsPerPC: cons, QtyWoScope: &scope, IDQtyWoShell: appShell, IDQtyWoSize: appSize}
	return parsedMaterialListImportRow{row: row, request: request, priceText: priceText, consText: consText, priceNumeric: priceNumeric, consNumeric: consNumeric}
}

func validateMaterialListImportRows(ctx context.Context, repo materialListItemCreateRepository, targetID int32, rows []parsedMaterialListImportRow) *MaterialListExcelImportValidationError {
	v := &MaterialListExcelImportValidationError{}
	for i := range rows {
		r := &rows[i]
		if err := validateMaterialListItemSources(ctx, repo, targetID, nullableInt32Param(r.request.IDWoShell), nullableInt32Param(r.request.IDWoTrim)); err != nil {
			addFieldError(v, r.row, "SOURCE_ID", "invalid_reference", "source ID does not belong to the target work order")
		}
		if err := validateMaterialListItemApplicability(ctx, repo, targetID, materialListItemApplicability{Category: r.request.Category, QtyWoScope: r.request.QtyWoScope, IDQtyWoShell: r.request.IDQtyWoShell, IDQtyWoSize: r.request.IDQtyWoSize}); err != nil {
			addFieldError(v, r.row, "QTY_WO_SCOPE", "invalid_applicability", "applicability IDs are not valid for the target work order")
		}
		if r.request.ConsPerPC == nil {
			cons, err := materialListConsPerPCForCreate(ctx, repo, nil, nullableInt32Param(r.request.IDWoShell), nullableInt32Param(r.request.IDWoTrim))
			if err != nil {
				addFieldError(v, r.row, "CONS_PER_PC", "invalid_reference", "consumption prefill could not be resolved")
			} else {
				r.consText = canonicalNumeric(cons, 3)
				resolved := cons
				r.consNumeric = &resolved
			}
		}
		r.key = materialListImportKey(r.request, r.priceText, r.consText)
	}
	return v
}

func validateMaterialListImportDuplicates(rows []parsedMaterialListImportRow, existing []entity.ListMaterialListItemsByMLRow, v *MaterialListExcelImportValidationError) {
	seen := map[string]int{}
	for _, row := range rows {
		if first, ok := seen[row.key]; ok {
			addFieldError(v, row.row, "ITEM", "duplicate_in_file", fmt.Sprintf("duplicates row %d", first))
		} else {
			seen[row.key] = row.row
		}
	}
	existingKeys := map[string]struct{}{}
	for _, item := range existing {
		category := nullableTextPtr(item.Category)
		scope := nullableTextPtr(item.QtyWoScope)
		req := model.CreateMaterialListItemBody{Item: item.Item, Description: item.Description, Qty: item.Qty, Unit: item.Unit, IDWoShell: nullableInt32PgTypePtr(item.IDWoShell), IDWoTrim: nullableInt32PgTypePtr(item.IDWoTrim), Category: category, QtyWoScope: scope, IDQtyWoShell: nullableInt32PgTypePtr(item.IDQtyWoShell), IDQtyWoSize: nullableInt32PgTypePtr(item.IDQtyWoSize)}
		existingKeys[materialListImportKey(req, canonicalNumeric(item.EstPrice, 2), canonicalNumeric(item.ConsPerPc, 3))] = struct{}{}
	}
	for _, row := range rows {
		if _, ok := existingKeys[row.key]; ok {
			addFieldError(v, row.row, "ITEM", "duplicate_existing_item", "matches an existing material list item")
		}
	}
}

func materialListImportKey(r model.CreateMaterialListItemBody, price, cons string) string {
	part := func(v *int32) materialListImportKeyPart {
		if v == nil {
			return materialListImportKeyPart{}
		}
		return materialListImportKeyPart{valid: true, value: strconv.FormatInt(int64(*v), 10)}
	}
	category, scope := materialListImportKeyPart{}, materialListImportKeyPart{}
	if r.Category != nil {
		category = materialListImportKeyPart{valid: true, value: *r.Category}
	}
	if r.QtyWoScope != nil {
		scope = materialListImportKeyPart{valid: true, value: *r.QtyWoScope}
	}
	return materialListImportCanonicalKey([]materialListImportKeyPart{
		category,
		{valid: true, value: r.Item},
		{valid: true, value: r.Description},
		{valid: true, value: strconv.FormatInt(int64(r.Qty), 10)},
		{valid: true, value: r.Unit},
		{valid: true, value: price},
		part(r.IDWoShell),
		part(r.IDWoTrim),
		{valid: cons != "", value: cons},
		scope,
		part(r.IDQtyWoShell),
		part(r.IDQtyWoSize),
	})
}

type materialListImportKeyPart struct {
	valid bool
	value string
}

// Length-prefixed fields preserve nil distinctly from blank and cannot collide
// even when user text contains delimiters or control characters.
func materialListImportCanonicalKey(parts []materialListImportKeyPart) string {
	var key strings.Builder
	for _, part := range parts {
		if !part.valid {
			key.WriteString("-1:")
			continue
		}
		key.WriteString(strconv.Itoa(len(part.value)))
		key.WriteByte(':')
		key.WriteString(part.value)
	}
	return key.String()
}

func parseQty(s string) (int32, bool) {
	if !regexp.MustCompile(`^[0-9]+$`).MatchString(s) {
		return 0, false
	}
	n, err := strconv.ParseInt(s, 10, 32)
	return int32(n), err == nil
}
func parsePositiveID(s string) (int32, bool) {
	if !regexp.MustCompile(`^[1-9][0-9]*$`).MatchString(s) {
		return 0, true
	}
	n, err := strconv.ParseInt(s, 10, 32)
	return int32(n), err != nil
}
func parseDecimal(s string, precision, scale int) (float64, string, string) {
	if s == "" {
		return 0, "", "required"
	}
	if !decimalLiteral.MatchString(s) {
		return 0, "", "invalid_decimal"
	}
	parts := strings.SplitN(s, ".", 2)
	frac := 0
	if len(parts) == 2 {
		frac = len(parts[1])
	}
	digits := strings.TrimLeft(parts[0], "0")
	if digits == "" {
		digits = "0"
	}
	if frac > scale || len(digits)+frac > precision {
		return 0, "", "invalid_precision"
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, "", "invalid_decimal"
	}
	return n, decimalFixed(s, scale), ""
}

func numericFromDecimal(s string) pgtype.Numeric {
	var numeric pgtype.Numeric
	_ = numeric.Scan(s)
	return numeric
}
func decimalFixed(s string, scale int) string {
	parts := strings.SplitN(s, ".", 2)
	whole := strings.TrimLeft(parts[0], "0")
	if whole == "" {
		whole = "0"
	}
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	return whole + "." + frac + strings.Repeat("0", scale-len(frac))
}
func canonicalNumeric(n pgtype.Numeric, scale int) string {
	if !n.Valid || n.Int == nil {
		return ""
	}
	value := new(big.Int).Set(n.Int)
	sign := ""
	if value.Sign() < 0 {
		sign = "-"
		value.Abs(value)
	}
	digits := value.String()
	exp := n.Exp
	if exp >= 0 {
		return sign + digits + strings.Repeat("0", int(exp)) + "." + strings.Repeat("0", scale)
	}
	places := int(-exp)
	if len(digits) <= places {
		digits = strings.Repeat("0", places-len(digits)+1) + digits
	}
	whole := digits[:len(digits)-places]
	frac := digits[len(digits)-places:]
	if len(frac) > scale {
		frac = frac[:scale]
	}
	return sign + whole + "." + frac + strings.Repeat("0", scale-len(frac))
}
func mustRows(book *excelize.File, sheet string) [][]string {
	rows, err := book.GetRows(sheet)
	if err != nil {
		return nil
	}
	return rows
}
func value(book *excelize.File, sheet, cell string) string {
	v, _ := book.GetCellValue(sheet, cell)
	return v
}
func fixedRow(row []string, width int) []string {
	out := make([]string, width)
	copy(out, row)
	return out
}
func rowHasText(row []string) bool {
	for _, v := range row {
		if strings.TrimSpace(v) != "" {
			return true
		}
	}
	return false
}
func addFieldError(v *MaterialListExcelImportValidationError, row int, column, code, message string) {
	addImportError(v, MaterialListExcelImportError{Row: row, Column: column, Code: code, Message: message})
}
func addImportError(v *MaterialListExcelImportValidationError, e MaterialListExcelImportError) {
	v.ErrorCount++
	if len(v.Errors) < materialListImportMaxRowErrors {
		v.Errors = append(v.Errors, e)
	}
}
func validationResult(v *MaterialListExcelImportValidationError) *MaterialListExcelImportResult {
	return &MaterialListExcelImportResult{Errors: v.Errors, ErrorCount: v.ErrorCount}
}
func importDomainError(code, message string) error {
	return &MaterialListExcelImportValidationError{Errors: []MaterialListExcelImportError{{Code: code, Message: message}}, ErrorCount: 1}
}
