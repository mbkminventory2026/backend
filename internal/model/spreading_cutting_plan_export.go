package model

import "time"

// SpreadingCuttingPlanExport is the authoritative renderer-neutral document
// composition. Nil derived or received values mean unavailable; numeric zero
// is retained when it is explicitly persisted or derived from persisted data.
type SpreadingCuttingPlanExport struct {
	Header          SpreadingCuttingPlanExportHeader
	Sizes           []SpreadingCuttingPlanExportSize
	Shells          []SpreadingCuttingPlanExportShell
	AllowanceHeader string
	Totals          SpreadingCuttingPlanExportTotals
}

type SpreadingCuttingPlanExportHeader struct {
	SpreadingCuttingPlanID int32
	DocumentNumber         string
	WorkOrderID            int32
	Buyer                  string
	EffectiveDate          time.Time
	Style                  string
	Model                  string
	Colors                 []string
	RatioPO                *string
}

type SpreadingCuttingPlanExportSize struct {
	SizeID int32
	Label  string
}

type SpreadingCuttingPlanExportShell struct {
	ShellID        int32
	Color          string
	OrderQty       []int64
	OrderTotal     int64
	ReceivedActual *float64
	Ratios         []SpreadingCuttingPlanExportRatio
	Subtotal       SpreadingCuttingPlanExportShellSubtotal
}

type SpreadingCuttingPlanExportRatio struct {
	RatioID              int32
	ComponentID          int32
	ComponentName        string
	Sequence             int
	Cons                 float64
	AllowancePercent     float64
	AllowanceConsumption float64
	ConsPerPC            float64
	RatioPlan            []*int32
	PlanSpreadingGelaran float64
	PlannedQty           []*int64
	PlannedTotal         int64
	RollQty              int32
	JoinedRoll           int32
	SambunganYard        float64
	Reject               float64
	FabricWidth          float64
	SubConsumption       float64
	TotalConsumption     float64
	RunningBalance       *float64
	Note                 string
}

type SpreadingCuttingPlanExportShellSubtotal struct {
	PlannedQty       []*int64
	PlannedTotal     int64
	SubConsumption   float64
	SambunganYard    float64
	TotalConsumption float64
	Reject           float64
	Balance          *float64
}

type SpreadingCuttingPlanExportTotals struct {
	OrderQty     []int64
	OrderTotal   int64
	PlannedQty   []*int64
	PlannedTotal int64
	BalanceQty   []*int64
	BalanceTotal int64
}
