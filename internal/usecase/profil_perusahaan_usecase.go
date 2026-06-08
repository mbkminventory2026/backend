package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

var (
	ErrProfilPerusahaanNotFound      = errors.New("profil perusahaan not found")
	ErrProfilPerusahaanAlreadyExists = errors.New("profil perusahaan data already exists")
)

type ProfilPerusahaanUseCase struct {
	repo entity.Querier
}

func NewProfilPerusahaanUseCase(repo entity.Querier) (*ProfilPerusahaanUseCase, error) {
	if repo == nil {
		return nil, errors.New("repository is required")
	}
	return &ProfilPerusahaanUseCase{repo: repo}, nil
}

func (u *ProfilPerusahaanUseCase) GetProfilPerusahaan(ctx context.Context) (model.ProfilPerusahaanResponse, error) {
	item, err := u.repo.GetProfilPerusahaan(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ProfilPerusahaanResponse{}, ErrProfilPerusahaanNotFound
		}
		return model.ProfilPerusahaanResponse{}, err
	}

	return model.ProfilPerusahaanResponse{
		ID:        item.IDProfilPerusahaan,
		Nama:      item.Nama,
		Alamat:    item.Alamat,
		Email:     item.Email,
		NoTelp:    item.NoTelp,
		About:     item.About,
		Logo:      item.Logo,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *ProfilPerusahaanUseCase) GetProfilPerusahaanByID(ctx context.Context, id int32) (model.ProfilPerusahaanResponse, error) {
	item, err := u.repo.GetProfilPerusahaanByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ProfilPerusahaanResponse{}, ErrProfilPerusahaanNotFound
		}
		return model.ProfilPerusahaanResponse{}, err
	}

	return model.ProfilPerusahaanResponse{
		ID:        item.IDProfilPerusahaan,
		Nama:      item.Nama,
		Alamat:    item.Alamat,
		Email:     item.Email,
		NoTelp:    item.NoTelp,
		About:     item.About,
		Logo:      item.Logo,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *ProfilPerusahaanUseCase) CreateProfilPerusahaan(ctx context.Context, req model.CreateProfilPerusahaanRequest) (model.ProfilPerusahaanResponse, error) {
	if _, err := u.repo.GetProfilPerusahaan(ctx); err == nil {
		return model.ProfilPerusahaanResponse{}, ErrProfilPerusahaanAlreadyExists
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return model.ProfilPerusahaanResponse{}, fmt.Errorf("check existing profil perusahaan: %w", err)
	}

	item, err := u.repo.CreateProfilPerusahaan(ctx, entity.CreateProfilPerusahaanParams{
		Nama:   req.Nama,
		Alamat: req.Alamat,
		Email:  req.Email,
		NoTelp: req.NoTelp,
		About:  req.About,
		Logo:   req.Logo,
	})
	if err != nil {
		return model.ProfilPerusahaanResponse{}, u.mapConflict(err)
	}

	return model.ProfilPerusahaanResponse{
		ID:        item.IDProfilPerusahaan,
		Nama:      item.Nama,
		Alamat:    item.Alamat,
		Email:     item.Email,
		NoTelp:    item.NoTelp,
		About:     item.About,
		Logo:      item.Logo,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *ProfilPerusahaanUseCase) UpdateProfilPerusahaan(ctx context.Context, id int32, req model.UpdateProfilPerusahaanRequest) (model.ProfilPerusahaanResponse, error) {
	item, err := u.repo.UpdateProfilPerusahaan(ctx, entity.UpdateProfilPerusahaanParams{
		IDProfilPerusahaan: id,
		Nama:                req.Nama,
		Alamat:              req.Alamat,
		Email:               req.Email,
		NoTelp:              req.NoTelp,
		About:               req.About,
		Logo:                req.Logo,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ProfilPerusahaanResponse{}, ErrProfilPerusahaanNotFound
		}
		return model.ProfilPerusahaanResponse{}, u.mapConflict(err)
	}

	return model.ProfilPerusahaanResponse{
		ID:        item.IDProfilPerusahaan,
		Nama:      item.Nama,
		Alamat:    item.Alamat,
		Email:     item.Email,
		NoTelp:    item.NoTelp,
		About:     item.About,
		Logo:      item.Logo,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *ProfilPerusahaanUseCase) DeleteProfilPerusahaan(ctx context.Context, id int32) error {
	affected, err := u.repo.DeleteProfilPerusahaan(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrProfilPerusahaanNotFound
	}
	return nil
}

func (u *ProfilPerusahaanUseCase) mapConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrProfilPerusahaanAlreadyExists
	}
	return err
}
