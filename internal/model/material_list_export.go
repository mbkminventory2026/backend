package model

import "time"

// MaterialListExport is the authoritative in-memory input for the future
// Material List XLSX renderer. Nil numeric fields mean unavailable, not zero.
type MaterialListExport struct {
	Header        MaterialListExportHeader
	DetailQty     MaterialListExportDetailQty
	SJDates       []time.Time
	ReceivedDates []time.Time
	MaterialRows  []MaterialListExportRow
}

type MaterialListExportHeader struct {
	MaterialListID   int32
	MaterialListName string
	WorkOrderID      int32
	Buyer            string
	Model            string
	Style            string
	WorkOrderQty     int64
	FobCmt           bool
	Delivery         time.Time
}

type MaterialListExportDetailQty struct {
	Sizes     []MaterialListExportSize
	Shells    []MaterialListExportShellQty
	Summaries []MaterialListExportSizeSummary
}

type MaterialListExportSize struct {
	ID   int32
	Name string
}

// Values are aligned to DetailQty.Sizes. A nil value keeps a genuinely absent
// shell-size cell blank for the renderer.
type MaterialListExportShellQty struct {
	ShellID     int32
	Color       string
	Description string
	Values      []MaterialListExportShellSizeQty
}

type MaterialListExportShellSizeQty struct {
	SizeID     int32
	Order      *int64
	MarkerPlan *int64
	ActualCut  *int64
}

type MaterialListExportSizeSummary struct {
	SizeID      int32
	OrderTotal  *int64
	RMSTotal    *int64
	TotalActCut *int64
	Balance     *int64
}

type MaterialListCuttingSource string

const (
	MaterialListCuttingNone    MaterialListCuttingSource = "NONE"
	MaterialListCuttingActual  MaterialListCuttingSource = "ACTUAL"
	MaterialListCuttingCutPlan MaterialListCuttingSource = "CUTPLAN"
)

type MaterialListExportRow struct {
	// MaterialListItemID and IDs below are composition keys only. A renderer
	// must select display fields explicitly and must never print these by default.
	MaterialListItemID int32
	Item               string
	Description        string
	Category           *string
	Unit               string
	ConsPerPC          *float64
	QtyWoScope         *string
	Applicability      MaterialListExportApplicability
	Source             MaterialListExportMaterialSource

	QtyWO              *int64
	Total              *float64
	SJByDate           map[time.Time]int64
	SJTotal            *int64
	ReceivedByDate     map[time.Time]int64
	ReceivedTotal      *int64
	BalanceReceived    *int64
	CuttingSource      MaterialListCuttingSource
	CuttingQty         *int64
	ActualConsumption  *float64
	RejectRetur        *int64
	ReconciliationNote *string
	FinalBalance       *float64
}

type MaterialListExportApplicability struct {
	ShellID    *int32
	ShellColor *string
	SizeID     *int32
	SizeName   *string
}

type MaterialListExportMaterialSource struct {
	ShellID          *int32
	ShellDescription *string
	ShellColor       *string
	TrimID           *int32
	TrimItem         *string
	TrimDescription  *string
	TrimColor        *string
	AllowancePercent *int32
}
