package model

type ListQueryFilter struct {
	Page     int32
	Limit    int32
	Offset   int32
	Search   string
	SortBy   string
	SortDesc bool
}
