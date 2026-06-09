package usecase

import (
	"context"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

type ProductionMasterUseCase struct {
	q *entity.Queries
}

func NewProductionMasterUseCase(q *entity.Queries) (*ProductionMasterUseCase, error) {
	return &ProductionMasterUseCase{
		q: q,
	}, nil
}

// PRODUCTION LINE

func (uc *ProductionMasterUseCase) GetProductionLineByID(ctx context.Context, id int32) (model.ProductionLineResponse, error) {
	item, err := uc.q.GetProductionLineByID(ctx, id)
	if err != nil {
		return model.ProductionLineResponse{}, err
	}
	return model.ProductionLineResponse{
		IDProductionLine: item.IDProductionLine,
		Name:             item.Name,
		CreatedAt:        item.CreatedAt.Time,
	}, nil
}

func (uc *ProductionMasterUseCase) ListProductionLines(ctx context.Context, filter model.ListQueryFilter) ([]model.ProductionLineResponse, int64, error) {
	arg := entity.ListProductionLinesParams{
		SearchTerm: filter.Search,
		SortBy:     filter.SortBy,
		SortDesc:   filter.SortDesc,
		PageLimit:  filter.Limit,
		PageOffset: filter.Offset,
	}

	items, err := uc.q.ListProductionLines(ctx, arg)
	if err != nil {
		return nil, 0, err
	}

	total, err := uc.q.CountProductionLines(ctx, filter.Search)
	if err != nil {
		return nil, 0, err
	}

	res := make([]model.ProductionLineResponse, 0, len(items))
	for _, i := range items {
		res = append(res, model.ProductionLineResponse{
			IDProductionLine: i.IDProductionLine,
			Name:             i.Name,
			CreatedAt:        i.CreatedAt.Time,
		})
	}
	return res, total, nil
}

func (uc *ProductionMasterUseCase) CreateProductionLine(ctx context.Context, req model.CreateProductionLineRequest) (model.ProductionLineResponse, error) {
	item, err := uc.q.CreateProductionLine(ctx, req.Name)
	if err != nil {
		return model.ProductionLineResponse{}, err
	}
	return model.ProductionLineResponse{
		IDProductionLine: item.IDProductionLine,
		Name:             item.Name,
		CreatedAt:        item.CreatedAt.Time,
	}, nil
}

func (uc *ProductionMasterUseCase) UpdateProductionLine(ctx context.Context, id int32, req model.UpdateProductionLineRequest) (model.ProductionLineResponse, error) {
	arg := entity.UpdateProductionLineParams{
		IDProductionLine: id,
		Name:             req.Name,
	}
	item, err := uc.q.UpdateProductionLine(ctx, arg)
	if err != nil {
		return model.ProductionLineResponse{}, err
	}
	return model.ProductionLineResponse{
		IDProductionLine: item.IDProductionLine,
		Name:             item.Name,
		CreatedAt:        item.CreatedAt.Time,
	}, nil
}

func (uc *ProductionMasterUseCase) DeleteProductionLine(ctx context.Context, id int32) error {
	_, err := uc.q.DeleteProductionLine(ctx, id)
	return err
}


// PRODUCTION STATUS PLAN

func (uc *ProductionMasterUseCase) GetProductionStatusPlanByID(ctx context.Context, id int32) (model.ProductionStatusPlanResponse, error) {
	item, err := uc.q.GetProductionStatusPlanByID(ctx, id)
	if err != nil {
		return model.ProductionStatusPlanResponse{}, err
	}
	return model.ProductionStatusPlanResponse{
		IDProductionStatusPlan: item.IDProductionStatusPlan,
		Name:                   item.Name,
		CreatedAt:              item.CreatedAt.Time,
	}, nil
}

func (uc *ProductionMasterUseCase) ListProductionStatusPlans(ctx context.Context, filter model.ListQueryFilter) ([]model.ProductionStatusPlanResponse, int64, error) {
	arg := entity.ListProductionStatusPlansParams{
		SearchTerm: filter.Search,
		SortBy:     filter.SortBy,
		SortDesc:   filter.SortDesc,
		PageLimit:  filter.Limit,
		PageOffset: filter.Offset,
	}

	items, err := uc.q.ListProductionStatusPlans(ctx, arg)
	if err != nil {
		return nil, 0, err
	}

	total, err := uc.q.CountProductionStatusPlans(ctx, filter.Search)
	if err != nil {
		return nil, 0, err
	}

	res := make([]model.ProductionStatusPlanResponse, 0, len(items))
	for _, i := range items {
		res = append(res, model.ProductionStatusPlanResponse{
			IDProductionStatusPlan: i.IDProductionStatusPlan,
			Name:                   i.Name,
			CreatedAt:              i.CreatedAt.Time,
		})
	}
	return res, total, nil
}

func (uc *ProductionMasterUseCase) CreateProductionStatusPlan(ctx context.Context, req model.CreateProductionStatusPlanRequest) (model.ProductionStatusPlanResponse, error) {
	item, err := uc.q.CreateProductionStatusPlan(ctx, req.Name)
	if err != nil {
		return model.ProductionStatusPlanResponse{}, err
	}
	return model.ProductionStatusPlanResponse{
		IDProductionStatusPlan: item.IDProductionStatusPlan,
		Name:                   item.Name,
		CreatedAt:              item.CreatedAt.Time,
	}, nil
}

func (uc *ProductionMasterUseCase) UpdateProductionStatusPlan(ctx context.Context, id int32, req model.UpdateProductionStatusPlanRequest) (model.ProductionStatusPlanResponse, error) {
	arg := entity.UpdateProductionStatusPlanParams{
		IDProductionStatusPlan: id,
		Name:                   req.Name,
	}
	item, err := uc.q.UpdateProductionStatusPlan(ctx, arg)
	if err != nil {
		return model.ProductionStatusPlanResponse{}, err
	}
	return model.ProductionStatusPlanResponse{
		IDProductionStatusPlan: item.IDProductionStatusPlan,
		Name:                   item.Name,
		CreatedAt:              item.CreatedAt.Time,
	}, nil
}

func (uc *ProductionMasterUseCase) DeleteProductionStatusPlan(ctx context.Context, id int32) error {
	_, err := uc.q.DeleteProductionStatusPlan(ctx, id)
	return err
}
