package usecase

import (
	"context"
	"errors"
	"time"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

var (
	ErrMasterDataNotFound = errors.New("master data not found")
)

type MasterDataUseCase struct {
	repo entity.Querier
}

func NewMasterDataUseCase(repo entity.Querier) (*MasterDataUseCase, error) {
	if repo == nil {
		return nil, errors.New("repository is required")
	}
	return &MasterDataUseCase{repo: repo}, nil
}

// DEPARTEMEN
func (u *MasterDataUseCase) ListDepartemen(ctx context.Context) ([]model.DepartemenResponse, error) {
	items, err := u.repo.ListDepartemen(ctx)
	if err != nil { return nil, err }
	
	res := make([]model.DepartemenResponse, 0, len(items))
	for _, i := range items {
		res = append(res, model.DepartemenResponse{
			ID: i.IDDepartemen,
			Nama: i.NamaDepartemen,
			CreatedAt: i.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	return res, nil
}

func (u *MasterDataUseCase) CreateDepartemen(ctx context.Context, req model.CreateDepartemenRequest) (model.DepartemenResponse, error) {
	item, err := u.repo.CreateDepartemen(ctx, req.NamaDepartemen)
	if err != nil { return model.DepartemenResponse{}, err }
	
	return model.DepartemenResponse{
		ID: item.IDDepartemen,
		Nama: item.NamaDepartemen,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) UpdateDepartemen(ctx context.Context, id int32, req model.UpdateDepartemenRequest) (model.DepartemenResponse, error) {
	item, err := u.repo.UpdateDepartemen(ctx, entity.UpdateDepartemenParams{
		IDDepartemen: id,
		NamaDepartemen: req.NamaDepartemen,
	})
	if err != nil { return model.DepartemenResponse{}, err }
	
	return model.DepartemenResponse{
		ID: item.IDDepartemen,
		Nama: item.NamaDepartemen,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) DeleteDepartemen(ctx context.Context, id int32) error {
	affected, err := u.repo.DeleteDepartemen(ctx, id)
	if err != nil { return err }
	if affected == 0 { return ErrMasterDataNotFound }
	return nil
}

// JENIS BARANG
func (u *MasterDataUseCase) ListJenisBarang(ctx context.Context) ([]model.JenisBarangResponse, error) {
	items, err := u.repo.ListJenisBarang(ctx)
	if err != nil { return nil, err }
	
	res := make([]model.JenisBarangResponse, 0, len(items))
	for _, i := range items {
		res = append(res, model.JenisBarangResponse{
			ID: i.IDJenisBarang,
			Nama: i.NamaJenisBarang,
			Kode: i.Kode,
			CreatedAt: i.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	return res, nil
}

func (u *MasterDataUseCase) CreateJenisBarang(ctx context.Context, req model.CreateJenisBarangRequest) (model.JenisBarangResponse, error) {
	item, err := u.repo.CreateJenisBarang(ctx, entity.CreateJenisBarangParams{
		NamaJenisBarang: req.NamaJenisBarang,
		Kode:            req.Kode,
	})
	if err != nil { return model.JenisBarangResponse{}, err }
	
	return model.JenisBarangResponse{
		ID: item.IDJenisBarang,
		Nama: item.NamaJenisBarang,
		Kode: item.Kode,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) UpdateJenisBarang(ctx context.Context, id int32, req model.UpdateJenisBarangRequest) (model.JenisBarangResponse, error) {
	item, err := u.repo.UpdateJenisBarang(ctx, entity.UpdateJenisBarangParams{
		IDJenisBarang: id,
		NamaJenisBarang: req.NamaJenisBarang,
		Kode: req.Kode,
	})
	if err != nil { return model.JenisBarangResponse{}, err }
	
	return model.JenisBarangResponse{
		ID: item.IDJenisBarang,
		Nama: item.NamaJenisBarang,
		Kode: item.Kode,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) DeleteJenisBarang(ctx context.Context, id int32) error {
	affected, err := u.repo.DeleteJenisBarang(ctx, id)
	if err != nil { return err }
	if affected == 0 { return ErrMasterDataNotFound }
	return nil
}

// MITRA
func (u *MasterDataUseCase) ListMitra(ctx context.Context) ([]model.MitraResponse, error) {
	items, err := u.repo.ListMitra(ctx)
	if err != nil { return nil, err }
	
	res := make([]model.MitraResponse, 0, len(items))
	for _, i := range items {
		res = append(res, model.MitraResponse{
			ID: i.IDMitra,
			NamaPerusahaan: i.NamaPerusahaan,
			TipePerusahaan: i.TipePerusahaan,
			Email: i.Email,
			NoTelp: i.NoTelp,
			CreatedAt: i.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	return res, nil
}

func (u *MasterDataUseCase) CreateMitra(ctx context.Context, req model.CreateMitraRequest) (model.MitraResponse, error) {
	item, err := u.repo.CreateMitra(ctx, entity.CreateMitraParams{
		NamaPerusahaan: req.NamaPerusahaan,
		TipePerusahaan: req.TipePerusahaan,
		Email:          req.Email,
		NoTelp:         req.NoTelp,
		Alamat:         req.Alamat,
		Kota:           req.Kota,
		KodePos:        req.KodePos,
	})
	if err != nil { return model.MitraResponse{}, err }
	
	return model.MitraResponse{
		ID: item.IDMitra,
		NamaPerusahaan: item.NamaPerusahaan,
		TipePerusahaan: item.TipePerusahaan,
		Email: item.Email,
		NoTelp: item.NoTelp,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) UpdateMitra(ctx context.Context, id int32, req model.UpdateMitraRequest) (model.MitraResponse, error) {
	item, err := u.repo.UpdateMitra(ctx, entity.UpdateMitraParams{
		IDMitra: id,
		NamaPerusahaan: req.NamaPerusahaan,
		TipePerusahaan: req.TipePerusahaan,
		Email: req.Email,
		NoTelp: req.NoTelp,
		Alamat: req.Alamat,
		Kota: req.Kota,
		KodePos: req.KodePos,
	})
	if err != nil { return model.MitraResponse{}, err }
	
	return model.MitraResponse{
		ID: item.IDMitra,
		NamaPerusahaan: item.NamaPerusahaan,
		TipePerusahaan: item.TipePerusahaan,
		Email: item.Email,
		NoTelp: item.NoTelp,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) DeleteMitra(ctx context.Context, id int32) error {
	affected, err := u.repo.DeleteMitra(ctx, id)
	if err != nil { return err }
	if affected == 0 { return ErrMasterDataNotFound }
	return nil
}

// BARANG
func (u *MasterDataUseCase) ListBarang(ctx context.Context, limit, offset int32) ([]model.BarangResponse, error) {
	items, err := u.repo.ListBarang(ctx, entity.ListBarangParams{Limit: limit, Offset: offset})
	if err != nil { return nil, err }
	
	res := make([]model.BarangResponse, 0, len(items))
	for _, i := range items {
		res = append(res, model.BarangResponse{
			ID: i.IDBarang,
			Nama: i.NamaBarang,
			Kode: i.Kode,
			NamaPerusahaan: i.NamaPerusahaan,
			NamaJenisBarang: i.NamaJenisBarang,
			CreatedAt: i.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	return res, nil
}

func (u *MasterDataUseCase) CreateBarang(ctx context.Context, req model.CreateBarangRequest) (model.BarangResponse, error) {
	item, err := u.repo.CreateBarang(ctx, entity.CreateBarangParams{
		NamaBarang: req.NamaBarang,
		Kode: req.Kode,
		IDJenisBarang: req.IDJenisBarang,
		IDMitra: req.IDMitra,
	})
	if err != nil { return model.BarangResponse{}, err }
	
	return model.BarangResponse{
		ID: item.IDBarang,
		Nama: item.NamaBarang,
		Kode: item.Kode,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) UpdateBarang(ctx context.Context, id int32, req model.UpdateBarangRequest) (model.BarangResponse, error) {
	item, err := u.repo.UpdateBarang(ctx, entity.UpdateBarangParams{
		IDBarang: id,
		NamaBarang: req.NamaBarang,
		Kode: req.Kode,
		IDJenisBarang: req.IDJenisBarang,
		IDMitra: req.IDMitra,
	})
	if err != nil { return model.BarangResponse{}, err }
	
	return model.BarangResponse{
		ID: item.IDBarang,
		Nama: item.NamaBarang,
		Kode: item.Kode,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) DeleteBarang(ctx context.Context, id int32) error {
	affected, err := u.repo.DeleteBarang(ctx, id)
	if err != nil { return err }
	if affected == 0 { return ErrMasterDataNotFound }
	return nil
}

// HAK AKSES
func (u *MasterDataUseCase) ListHakAkses(ctx context.Context) ([]entity.HakAkse, error) {
	return u.repo.ListHakAkses(ctx)
}

// COMPANY
func (u *MasterDataUseCase) GetCompany(ctx context.Context) (model.CompanyResponse, error) {
	item, err := u.repo.GetCompany(ctx)
	if err != nil { return model.CompanyResponse{}, err }
	
	return model.CompanyResponse{
		ID: item.IDCompany,
		Nama: item.Nama,
		Alamat: item.Alamat,
		Email: item.Email,
		NoTelp: item.NoTelp,
		About: item.About,
		Logo: item.Logo,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) UpdateCompany(ctx context.Context, id int32, req model.UpdateCompanyRequest) (model.CompanyResponse, error) {
	item, err := u.repo.UpdateCompany(ctx, entity.UpdateCompanyParams{
		IDCompany: id,
		Nama: req.Nama,
		Alamat: req.Alamat,
		Email: req.Email,
		NoTelp: req.NoTelp,
		About: req.About,
		Logo: req.Logo,
	})
	if err != nil { return model.CompanyResponse{}, err }
	
	return model.CompanyResponse{
		ID: item.IDCompany,
		Nama: item.Nama,
		Alamat: item.Alamat,
		Email: item.Email,
		NoTelp: item.NoTelp,
		About: item.About,
		Logo: item.Logo,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}
