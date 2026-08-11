package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

var (
	ErrSpreadingCuttingPlanExportDataConflict = errors.New("spreading_cutting_plan_export_data_conflict")
	ErrSpreadingCuttingPlanExportUnavailable  = errors.New("spreading cutting plan export unavailable")
)

type spreadingCuttingPlanExportRepository interface {
	GetSpreadingCuttingPlanExportHeader(context.Context, int32) (entity.GetSpreadingCuttingPlanExportHeaderRow, error)
	ListSpreadingCuttingPlanExportShellSizes(context.Context, int32) ([]entity.ListSpreadingCuttingPlanExportShellSizesRow, error)
	ListSpreadingCuttingPlanExportRatios(context.Context, int32) ([]entity.ListSpreadingCuttingPlanExportRatiosRow, error)
	ListSpreadingCuttingPlanExportReceivedByShell(context.Context, int32) ([]entity.ListSpreadingCuttingPlanExportReceivedByShellRow, error)
}

// SpreadingCuttingPlanExportUseCase composes a fixed SCP snapshot without
// selecting or consulting Marker Plan, DACP, or cutting actuals.
type SpreadingCuttingPlanExportUseCase struct {
	repo spreadingCuttingPlanExportRepository
}

func NewSpreadingCuttingPlanExportUseCase(repo spreadingCuttingPlanExportRepository) (*SpreadingCuttingPlanExportUseCase, error) {
	if repo == nil {
		return nil, errors.New("spreading cutting plan export repository is required")
	}
	return &SpreadingCuttingPlanExportUseCase{repo: repo}, nil
}

func (u *SpreadingCuttingPlanExportUseCase) Compose(ctx context.Context, idSpreadingCuttingPlan int32) (*model.SpreadingCuttingPlanExport, error) {
	header, err := u.repo.GetSpreadingCuttingPlanExportHeader(ctx, idSpreadingCuttingPlan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSpreadingCuttingPlanNotFound
		}
		return nil, spreadingCuttingPlanExportReadError("header", err)
	}
	shellSizes, err := u.repo.ListSpreadingCuttingPlanExportShellSizes(ctx, header.IDWo)
	if err != nil {
		return nil, spreadingCuttingPlanExportReadError("shell sizes", err)
	}
	ratios, err := u.repo.ListSpreadingCuttingPlanExportRatios(ctx, idSpreadingCuttingPlan)
	if err != nil {
		return nil, spreadingCuttingPlanExportReadError("ratios", err)
	}
	received, err := u.repo.ListSpreadingCuttingPlanExportReceivedByShell(ctx, header.IDWo)
	if err != nil {
		return nil, spreadingCuttingPlanExportReadError("received", err)
	}

	return composeSpreadingCuttingPlanExport(spreadingCuttingPlanExportSnapshot{
		header: header, shellSizes: shellSizes, ratios: ratios, received: received,
	})
}

func spreadingCuttingPlanExportReadError(part string, err error) error {
	return fmt.Errorf("%w: %s: %v", ErrSpreadingCuttingPlanExportUnavailable, part, err)
}

type spreadingCuttingPlanExportSnapshot struct {
	header     entity.GetSpreadingCuttingPlanExportHeaderRow
	shellSizes []entity.ListSpreadingCuttingPlanExportShellSizesRow
	ratios     []entity.ListSpreadingCuttingPlanExportRatiosRow
	received   []entity.ListSpreadingCuttingPlanExportReceivedByShellRow
}

type spreadingCuttingPlanExportShellSource struct {
	id        int32
	color     string
	sizeOrder []int32
	sizes     map[int32]spreadingCuttingPlanExportShellSizeSource
	bySizeID  map[int32]int32
}

type spreadingCuttingPlanExportShellSizeSource struct {
	id       int32
	sizeID   int32
	label    string
	orderQty int64
}

type spreadingCuttingPlanExportRatioSource struct {
	id              int32
	componentID     int32
	componentName   string
	shellID         int32
	cons            float64
	spread          float64
	allowance       float64
	rollQty         int32
	joinedRoll      int32
	reject          float64
	fabricWidth     float64
	note            string
	ratioPlanBySize map[int32]int32
}

func composeSpreadingCuttingPlanExport(snapshot spreadingCuttingPlanExportSnapshot) (*model.SpreadingCuttingPlanExport, error) {
	if !snapshot.header.TanggalEfektif.Valid {
		return nil, spreadingCuttingPlanExportConflict("effective date is unavailable")
	}

	shells, err := spreadingCuttingPlanExportShellSources(snapshot.shellSizes)
	if err != nil {
		return nil, err
	}
	ratioSources, shellOrder, err := spreadingCuttingPlanExportRatioSources(snapshot.header.IDWo, shells, snapshot.ratios)
	if err != nil {
		return nil, err
	}
	if len(shellOrder) == 0 {
		return nil, spreadingCuttingPlanExportConflict("spreading cutting plan has no ratio shells")
	}
	receivedByShell, err := spreadingCuttingPlanExportReceived(snapshot.header.IDWo, snapshot.received)
	if err != nil {
		return nil, err
	}

	sizeOrder, ratioPO, err := spreadingCuttingPlanExportRatioPO(shellOrder, shells)
	if err != nil {
		return nil, err
	}

	export := &model.SpreadingCuttingPlanExport{
		Header: model.SpreadingCuttingPlanExportHeader{
			SpreadingCuttingPlanID: snapshot.header.IDSpreadingCuttingPlan,
			DocumentNumber:         strings.TrimSpace(snapshot.header.NoDokumen),
			WorkOrderID:            snapshot.header.IDWo,
			Buyer:                  strings.TrimSpace(snapshot.header.Buyer),
			EffectiveDate:          snapshot.header.TanggalEfektif.Time,
			Style:                  strings.TrimSpace(snapshot.header.Style),
			Model:                  strings.TrimSpace(snapshot.header.Model),
			RatioPO:                ratioPO,
		},
	}

	firstShell := shells[shellOrder[0]]
	for _, sizeID := range sizeOrder {
		size := firstShell.sizes[firstShell.bySizeID[sizeID]]
		export.Sizes = append(export.Sizes, model.SpreadingCuttingPlanExportSize{SizeID: sizeID, Label: size.label})
	}

	allowanceSet := false
	commonAllowance := float64(0)
	allowanceMixed := false
	ratioByShell := make(map[int32][]spreadingCuttingPlanExportRatioSource, len(shellOrder))
	for _, ratio := range ratioSources {
		ratioByShell[ratio.shellID] = append(ratioByShell[ratio.shellID], ratio)
		if !allowanceSet {
			commonAllowance, allowanceSet = ratio.allowance, true
		} else if ratio.allowance != commonAllowance {
			allowanceMixed = true
		}
	}
	if allowanceSet && !allowanceMixed {
		export.AllowanceHeader = strconv.FormatFloat(commonAllowance, 'f', -1, 64) + "%"
	} else {
		export.AllowanceHeader = "ALLOW."
	}

	documentPlanned := make([]*int64, len(sizeOrder))
	for _, shellID := range shellOrder {
		source := shells[shellID]
		shell := model.SpreadingCuttingPlanExportShell{
			ShellID: shellID,
			Color:   source.color,
		}
		export.Header.Colors = append(export.Header.Colors, source.color)
		if received, ok := receivedByShell[shellID]; ok {
			receivedValue := float64(received)
			shell.ReceivedActual = &receivedValue
		}
		for _, sizeID := range sizeOrder {
			size := source.sizes[source.bySizeID[sizeID]]
			shell.OrderQty = append(shell.OrderQty, size.orderQty)
			shell.OrderTotal += size.orderQty
		}

		var runningBalance *float64
		if shell.ReceivedActual != nil {
			value := *shell.ReceivedActual
			runningBalance = &value
		}
		for ratioIndex, ratioSource := range ratioByShell[shellID] {
			ratio := model.SpreadingCuttingPlanExportRatio{
				RatioID:              ratioSource.id,
				ComponentID:          ratioSource.componentID,
				ComponentName:        ratioSource.componentName,
				Sequence:             ratioIndex + 1,
				Cons:                 ratioSource.cons,
				AllowancePercent:     ratioSource.allowance,
				PlanSpreadingGelaran: ratioSource.spread,
				RollQty:              ratioSource.rollQty,
				JoinedRoll:           ratioSource.joinedRoll,
				Reject:               ratioSource.reject,
				FabricWidth:          ratioSource.fabricWidth,
				Note:                 strings.TrimSpace(ratioSource.note),
			}
			ratio.AllowanceConsumption = ratio.Cons * ratio.AllowancePercent / 100
			ratio.ConsPerPC = ratio.Cons * (1 + ratio.AllowancePercent/100)
			for sizeIndex, sizeID := range sizeOrder {
				ratioPlan, ok := ratioSource.ratioPlanBySize[sizeID]
				if !ok {
					ratio.RatioPlan = append(ratio.RatioPlan, nil)
					ratio.PlannedQty = append(ratio.PlannedQty, nil)
					continue
				}
				ratioPlanValue := ratioPlan
				plannedValue := int64(math.Round(float64(ratioPlan) * ratio.PlanSpreadingGelaran))
				ratio.RatioPlan = append(ratio.RatioPlan, &ratioPlanValue)
				ratio.PlannedQty = append(ratio.PlannedQty, &plannedValue)
				ratio.PlannedTotal += plannedValue
				if shell.Subtotal.PlannedQty == nil {
					shell.Subtotal.PlannedQty = make([]*int64, len(sizeOrder))
				}
				if shell.Subtotal.PlannedQty[sizeIndex] == nil {
					zero := int64(0)
					shell.Subtotal.PlannedQty[sizeIndex] = &zero
				}
				*shell.Subtotal.PlannedQty[sizeIndex] += plannedValue
			}
			ratio.SubConsumption = float64(ratio.PlannedTotal) * ratio.ConsPerPC
			ratio.SambunganYard = float64(ratio.RollQty) * float64(ratio.JoinedRoll)
			ratio.TotalConsumption = ratio.SubConsumption + ratio.SambunganYard
			if runningBalance != nil {
				*runningBalance -= ratio.TotalConsumption + ratio.Reject
				value := *runningBalance
				ratio.RunningBalance = &value
			}
			shell.Subtotal.PlannedTotal += ratio.PlannedTotal
			shell.Subtotal.SubConsumption += ratio.SubConsumption
			shell.Subtotal.SambunganYard += ratio.SambunganYard
			shell.Subtotal.TotalConsumption += ratio.TotalConsumption
			shell.Subtotal.Reject += ratio.Reject
			shell.Ratios = append(shell.Ratios, ratio)
		}
		if runningBalance != nil {
			value := *runningBalance
			shell.Subtotal.Balance = &value
		}

		for sizeIndex, orderQty := range shell.OrderQty {
			export.Totals.OrderQty = appendOrAddInt64(export.Totals.OrderQty, sizeIndex, orderQty)
			export.Totals.OrderTotal += orderQty
			if sizeIndex < len(shell.Subtotal.PlannedQty) && shell.Subtotal.PlannedQty[sizeIndex] != nil {
				if documentPlanned[sizeIndex] == nil {
					zero := int64(0)
					documentPlanned[sizeIndex] = &zero
				}
				*documentPlanned[sizeIndex] += *shell.Subtotal.PlannedQty[sizeIndex]
			}
		}
		export.Totals.PlannedTotal += shell.Subtotal.PlannedTotal
		export.Shells = append(export.Shells, shell)
	}

	export.Totals.PlannedQty = documentPlanned
	export.Totals.BalanceQty = make([]*int64, len(sizeOrder))
	for index, planned := range documentPlanned {
		if planned == nil {
			continue
		}
		balance := *planned - export.Totals.OrderQty[index]
		export.Totals.BalanceQty[index] = &balance
	}
	export.Totals.BalanceTotal = export.Totals.PlannedTotal - export.Totals.OrderTotal

	return export, nil
}

func spreadingCuttingPlanExportShellSources(rows []entity.ListSpreadingCuttingPlanExportShellSizesRow) (map[int32]spreadingCuttingPlanExportShellSource, error) {
	shells := make(map[int32]spreadingCuttingPlanExportShellSource)
	shellSizeOwner := make(map[int32]int32)
	for _, row := range rows {
		shell, ok := shells[row.IDWoShell]
		if !ok {
			shell = spreadingCuttingPlanExportShellSource{
				id: row.IDWoShell, color: strings.TrimSpace(row.Color),
				sizes:    make(map[int32]spreadingCuttingPlanExportShellSizeSource),
				bySizeID: make(map[int32]int32),
			}
		} else if shell.color != strings.TrimSpace(row.Color) {
			return nil, spreadingCuttingPlanExportConflict("conflicting shell labels")
		}

		validCount := 0
		for _, valid := range []bool{row.IDWoShellSize.Valid, row.IDSize.Valid, row.NamaSize.Valid, row.OrderQty.Valid} {
			if valid {
				validCount++
			}
		}
		if validCount == 0 {
			shells[row.IDWoShell] = shell
			continue
		}
		if validCount != 4 || strings.TrimSpace(row.NamaSize.String) == "" {
			return nil, spreadingCuttingPlanExportConflict("incomplete work order shell size")
		}
		if owner, exists := shellSizeOwner[row.IDWoShellSize.Int32]; exists && owner != row.IDWoShell {
			return nil, spreadingCuttingPlanExportConflict("work order shell size belongs to multiple shells")
		}
		if _, exists := shell.sizes[row.IDWoShellSize.Int32]; exists {
			return nil, spreadingCuttingPlanExportConflict("duplicate work order shell size")
		}
		if _, exists := shell.bySizeID[row.IDSize.Int32]; exists {
			return nil, spreadingCuttingPlanExportConflict("duplicate master size in shell")
		}
		shellSizeOwner[row.IDWoShellSize.Int32] = row.IDWoShell
		shell.sizes[row.IDWoShellSize.Int32] = spreadingCuttingPlanExportShellSizeSource{
			id: row.IDWoShellSize.Int32, sizeID: row.IDSize.Int32,
			label: strings.TrimSpace(row.NamaSize.String), orderQty: int64(row.OrderQty.Int32),
		}
		shell.bySizeID[row.IDSize.Int32] = row.IDWoShellSize.Int32
		shell.sizeOrder = append(shell.sizeOrder, row.IDSize.Int32)
		shells[row.IDWoShell] = shell
	}
	return shells, nil
}

func spreadingCuttingPlanExportRatioSources(
	workOrderID int32,
	shells map[int32]spreadingCuttingPlanExportShellSource,
	rows []entity.ListSpreadingCuttingPlanExportRatiosRow,
) ([]spreadingCuttingPlanExportRatioSource, []int32, error) {
	ratioByID := make(map[int32]*spreadingCuttingPlanExportRatioSource)
	ratioOrder := make([]int32, 0)
	shellOrder := make([]int32, 0)
	seenShell := make(map[int32]struct{})

	for _, row := range rows {
		if row.RatioShellWoID != workOrderID {
			return nil, nil, spreadingCuttingPlanExportConflict("ratio shell is outside spreading cutting plan work order")
		}
		shell, ok := shells[row.RatioShellID]
		if !ok {
			return nil, nil, spreadingCuttingPlanExportConflict("ratio shell is not in spreading cutting plan work order")
		}
		ratio, exists := ratioByID[row.IDRatioSpreading]
		if !exists {
			cons, err := spreadingCuttingPlanExportNumeric(row.Cons, "cons")
			if err != nil {
				return nil, nil, err
			}
			spread, err := spreadingCuttingPlanExportNumeric(row.PlanSpreadingGelaran, "plan spreading")
			if err != nil {
				return nil, nil, err
			}
			allowance, err := spreadingCuttingPlanExportNumeric(row.Allowance, "allowance")
			if err != nil {
				return nil, nil, err
			}
			reject, err := spreadingCuttingPlanExportNumeric(row.Reject, "reject")
			if err != nil {
				return nil, nil, err
			}
			fabricWidth, err := spreadingCuttingPlanExportNumeric(row.LebarKain, "fabric width")
			if err != nil {
				return nil, nil, err
			}
			ratio = &spreadingCuttingPlanExportRatioSource{
				id: row.IDRatioSpreading, componentID: row.IDKomponenSpreading,
				componentName: strings.TrimSpace(row.NamaKomponen), shellID: row.RatioShellID,
				cons: cons, spread: spread, allowance: allowance, rollQty: row.RollQty,
				joinedRoll: row.SambunganRoll, reject: reject, fabricWidth: fabricWidth,
				note: row.Ket, ratioPlanBySize: make(map[int32]int32),
			}
			ratioByID[row.IDRatioSpreading] = ratio
			ratioOrder = append(ratioOrder, row.IDRatioSpreading)
			if _, seen := seenShell[row.RatioShellID]; !seen {
				seenShell[row.RatioShellID] = struct{}{}
				shellOrder = append(shellOrder, row.RatioShellID)
			}
		} else if ratio.shellID != row.RatioShellID || ratio.componentID != row.IDKomponenSpreading {
			return nil, nil, spreadingCuttingPlanExportConflict("ratio identity has conflicting parent relationships")
		}

		validCount := 0
		for _, valid := range []bool{
			row.IDRatioSizeSpreading.Valid, row.IDWoShellSize.Valid, row.RatioPlan.Valid,
			row.RatioSizeShellID.Valid, row.RatioSizeShellWoID.Valid, row.IDSize.Valid,
		} {
			if valid {
				validCount++
			}
		}
		if validCount == 0 {
			continue
		}
		if validCount != 6 {
			return nil, nil, spreadingCuttingPlanExportConflict("incomplete ratio size relationship")
		}
		if row.RatioSizeShellWoID.Int32 != workOrderID {
			return nil, nil, spreadingCuttingPlanExportConflict("ratio size shell is outside spreading cutting plan work order")
		}
		if row.RatioSizeShellID.Int32 != row.RatioShellID {
			return nil, nil, spreadingCuttingPlanExportConflict("ratio size shell does not belong to ratio shell")
		}
		shellSize, ok := shell.sizes[row.IDWoShellSize.Int32]
		if !ok || shellSize.sizeID != row.IDSize.Int32 {
			return nil, nil, spreadingCuttingPlanExportConflict("ratio size does not match exact work order shell size")
		}
		if _, duplicate := ratio.ratioPlanBySize[row.IDSize.Int32]; duplicate {
			return nil, nil, spreadingCuttingPlanExportConflict("duplicate ratio size for exact work order shell size")
		}
		ratio.ratioPlanBySize[row.IDSize.Int32] = row.RatioPlan.Int32
	}

	ratios := make([]spreadingCuttingPlanExportRatioSource, 0, len(ratioOrder))
	for _, ratioID := range ratioOrder {
		ratios = append(ratios, *ratioByID[ratioID])
	}
	return ratios, shellOrder, nil
}

func spreadingCuttingPlanExportReceived(workOrderID int32, rows []entity.ListSpreadingCuttingPlanExportReceivedByShellRow) (map[int32]int64, error) {
	received := make(map[int32]int64, len(rows))
	for _, row := range rows {
		if !row.IDWoShell.Valid || row.SourceShellWoID != workOrderID {
			return nil, spreadingCuttingPlanExportConflict("received material list item shell is outside work order")
		}
		if _, duplicate := received[row.IDWoShell.Int32]; duplicate {
			return nil, spreadingCuttingPlanExportConflict("duplicate received aggregate for shell")
		}
		received[row.IDWoShell.Int32] = row.ReceivedActual
	}
	return received, nil
}

func spreadingCuttingPlanExportRatioPO(
	shellOrder []int32,
	shells map[int32]spreadingCuttingPlanExportShellSource,
) ([]int32, *string, error) {
	first := shells[shellOrder[0]]
	if len(first.sizeOrder) == 0 {
		return nil, nil, spreadingCuttingPlanExportConflict("ratio shell has no work order sizes")
	}
	sizeOrder := append([]int32(nil), first.sizeOrder...)
	var common []int64
	commonAvailable := false

	for shellIndex, shellID := range shellOrder {
		shell := shells[shellID]
		if len(shell.sizeOrder) != len(sizeOrder) {
			return nil, nil, spreadingCuttingPlanExportConflict("ratio PO size sets differ between shells")
		}
		vector := make([]int64, len(sizeOrder))
		for index, sizeID := range sizeOrder {
			shellSizeID, ok := shell.bySizeID[sizeID]
			if !ok {
				return nil, nil, spreadingCuttingPlanExportConflict("ratio PO size sets differ between shells")
			}
			size := shell.sizes[shellSizeID]
			if strings.TrimSpace(size.label) != strings.TrimSpace(first.sizes[first.bySizeID[sizeID]].label) {
				return nil, nil, spreadingCuttingPlanExportConflict("master size labels conflict between shells")
			}
			vector[index] = size.orderQty
		}
		normalized, available := normalizeSpreadingCuttingPlanExportRatio(vector)
		if shellIndex == 0 {
			common, commonAvailable = normalized, available
			continue
		}
		if available != commonAvailable || !equalSpreadingCuttingPlanExportRatio(common, normalized) {
			return nil, nil, spreadingCuttingPlanExportConflict("normalized ratio PO differs between shells")
		}
	}
	if !commonAvailable {
		return sizeOrder, nil, nil
	}
	parts := make([]string, len(common))
	for index, value := range common {
		parts[index] = strconv.FormatInt(value, 10)
	}
	formatted := strings.Join(parts, "-")
	return sizeOrder, &formatted, nil
}

func normalizeSpreadingCuttingPlanExportRatio(values []int64) ([]int64, bool) {
	normalized := append([]int64(nil), values...)
	gcd := int64(0)
	for _, value := range values {
		if value == 0 {
			continue
		}
		if value < 0 {
			value = -value
		}
		gcd = spreadingCuttingPlanExportGCD(gcd, value)
	}
	if gcd == 0 {
		return normalized, false
	}
	for index, value := range normalized {
		if value != 0 {
			normalized[index] = value / gcd
		}
	}
	return normalized, true
}

func spreadingCuttingPlanExportGCD(left, right int64) int64 {
	for right != 0 {
		left, right = right, left%right
	}
	if left < 0 {
		return -left
	}
	return left
}

func equalSpreadingCuttingPlanExportRatio(left, right []int64) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func spreadingCuttingPlanExportNumeric(value pgtype.Numeric, field string) (float64, error) {
	result, err := value.Float64Value()
	if err != nil || !result.Valid {
		return 0, spreadingCuttingPlanExportConflict(field + " is unavailable")
	}
	return result.Float64, nil
}

func appendOrAddInt64(values []int64, index int, value int64) []int64 {
	for len(values) <= index {
		values = append(values, 0)
	}
	values[index] += value
	return values
}

func spreadingCuttingPlanExportConflict(message string) error {
	return fmt.Errorf("%w: %s", ErrSpreadingCuttingPlanExportDataConflict, message)
}
