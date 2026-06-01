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
	ErrMasterDataNotFound      = errors.New("master data not found")
	ErrMasterDataConflict      = errors.New("master data already exists")
	ErrMasterDataDuplicateCode = errors.New("master data code already exists")
	ErrCompanyAlreadyExists    = errors.New("company data already exists")

	departemenSortColumns  = buildSortWhitelist("created_at", "id_departemen", "nama_departemen")
	jenisBarangSortColumns = buildSortWhitelist("created_at", "id_jenis_barang", "kode", "nama_jenis_barang")
	mitraSortColumns       = buildSortWhitelist("created_at", "id_mitra", "nama_perusahaan", "email", "no_telp", "tipe_perusahaan")
	barangSortColumns      = buildSortWhitelist("created_at", "id_barang", "kode", "nama_barang", "nama_jenis_barang", "nama_perusahaan")
	hakAksesSortColumns    = buildSortWhitelist("created_at", "id_hak_akses", "nama_halaman")
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
func (u *MasterDataUseCase) GetDepartemenByID(ctx context.Context, id int32) (model.DepartemenResponse, error) {
	item, err := u.repo.GetDepartemenByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.DepartemenResponse{}, ErrMasterDataNotFound
		}
		return model.DepartemenResponse{}, err
	}

	return model.DepartemenResponse{
		ID:        item.IDDepartemen,
		Nama:      item.NamaDepartemen,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) ListDepartemen(ctx context.Context, filter model.ListQueryFilter) ([]model.DepartemenResponse, int64, error) {
	_, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter, "nama_departemen", false, departemenSortColumns)

	items, err := u.repo.ListDepartemen(ctx, entity.ListDepartemenParams{
		SearchTerm: search,
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := u.repo.CountDepartemen(ctx, search)
	if err != nil {
		return nil, 0, err
	}

	res := make([]model.DepartemenResponse, 0, len(items))
	for _, i := range items {
		res = append(res, model.DepartemenResponse{
			ID:        i.IDDepartemen,
			Nama:      i.NamaDepartemen,
			CreatedAt: i.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	return res, total, nil
}

func (u *MasterDataUseCase) CreateDepartemen(ctx context.Context, req model.CreateDepartemenRequest) (model.DepartemenResponse, error) {
	item, err := u.repo.CreateDepartemen(ctx, req.NamaDepartemen)
	if err != nil {
		return model.DepartemenResponse{}, mapMasterDataConflict(err)
	}

	return model.DepartemenResponse{
		ID:        item.IDDepartemen,
		Nama:      item.NamaDepartemen,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) UpdateDepartemen(ctx context.Context, id int32, req model.UpdateDepartemenRequest) (model.DepartemenResponse, error) {
	item, err := u.repo.UpdateDepartemen(ctx, entity.UpdateDepartemenParams{
		IDDepartemen:   id,
		NamaDepartemen: req.NamaDepartemen,
	})
	if err != nil {
		return model.DepartemenResponse{}, mapMasterDataConflict(err)
	}

	return model.DepartemenResponse{
		ID:        item.IDDepartemen,
		Nama:      item.NamaDepartemen,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) DeleteDepartemen(ctx context.Context, id int32) error {
	affected, err := u.repo.DeleteDepartemen(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrMasterDataNotFound
	}
	return nil
}

// JENIS BARANG
func (u *MasterDataUseCase) GetJenisBarangByID(ctx context.Context, id int32) (model.JenisBarangResponse, error) {
	item, err := u.repo.GetJenisBarangByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.JenisBarangResponse{}, ErrMasterDataNotFound
		}
		return model.JenisBarangResponse{}, err
	}

	return model.JenisBarangResponse{
		ID:        item.IDJenisBarang,
		Nama:      item.NamaJenisBarang,
		Kode:      item.Kode,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) ListJenisBarang(ctx context.Context, filter model.ListQueryFilter) ([]model.JenisBarangResponse, int64, error) {
	_, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter, "nama_jenis_barang", false, jenisBarangSortColumns)

	items, err := u.repo.ListJenisBarang(ctx, entity.ListJenisBarangParams{
		SearchTerm: search,
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := u.repo.CountJenisBarang(ctx, search)
	if err != nil {
		return nil, 0, err
	}

	res := make([]model.JenisBarangResponse, 0, len(items))
	for _, i := range items {
		res = append(res, model.JenisBarangResponse{
			ID:        i.IDJenisBarang,
			Nama:      i.NamaJenisBarang,
			Kode:      i.Kode,
			CreatedAt: i.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	return res, total, nil
}

func (u *MasterDataUseCase) CreateJenisBarang(ctx context.Context, req model.CreateJenisBarangRequest) (model.JenisBarangResponse, error) {
	item, err := u.repo.CreateJenisBarang(ctx, entity.CreateJenisBarangParams{
		NamaJenisBarang: req.NamaJenisBarang,
		Kode:            req.Kode,
	})
	if err != nil {
		return model.JenisBarangResponse{}, mapMasterDataConflict(err)
	}

	return model.JenisBarangResponse{
		ID:        item.IDJenisBarang,
		Nama:      item.NamaJenisBarang,
		Kode:      item.Kode,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) UpdateJenisBarang(ctx context.Context, id int32, req model.UpdateJenisBarangRequest) (model.JenisBarangResponse, error) {
	item, err := u.repo.UpdateJenisBarang(ctx, entity.UpdateJenisBarangParams{
		IDJenisBarang:   id,
		NamaJenisBarang: req.NamaJenisBarang,
		Kode:            req.Kode,
	})
	if err != nil {
		return model.JenisBarangResponse{}, mapMasterDataConflict(err)
	}

	return model.JenisBarangResponse{
		ID:        item.IDJenisBarang,
		Nama:      item.NamaJenisBarang,
		Kode:      item.Kode,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) DeleteJenisBarang(ctx context.Context, id int32) error {
	affected, err := u.repo.DeleteJenisBarang(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrMasterDataNotFound
	}
	return nil
}

// MITRA
func (u *MasterDataUseCase) GetMitraByID(ctx context.Context, id int32) (model.MitraResponse, error) {
	item, err := u.repo.GetMitraByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.MitraResponse{}, ErrMasterDataNotFound
		}
		return model.MitraResponse{}, err
	}

	return model.MitraResponse{
		ID:             item.IDMitra,
		NamaPerusahaan: item.NamaPerusahaan,
		TipePerusahaan: item.TipePerusahaan,
		Email:          item.Email,
		NoTelp:         item.NoTelp,
		CreatedAt:      item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) ListMitra(ctx context.Context, filter model.ListQueryFilter) ([]model.MitraResponse, int64, error) {
	_, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter, "nama_perusahaan", false, mitraSortColumns)

	items, err := u.repo.ListMitra(ctx, entity.ListMitraParams{
		SearchTerm: search,
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := u.repo.CountMitra(ctx, search)
	if err != nil {
		return nil, 0, err
	}

	res := make([]model.MitraResponse, 0, len(items))
	for _, i := range items {
		res = append(res, model.MitraResponse{
			ID:             i.IDMitra,
			NamaPerusahaan: i.NamaPerusahaan,
			TipePerusahaan: i.TipePerusahaan,
			Email:          i.Email,
			NoTelp:         i.NoTelp,
			CreatedAt:      i.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	return res, total, nil
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
	if err != nil {
		return model.MitraResponse{}, mapMasterDataConflict(err)
	}

	return model.MitraResponse{
		ID:             item.IDMitra,
		NamaPerusahaan: item.NamaPerusahaan,
		TipePerusahaan: item.TipePerusahaan,
		Email:          item.Email,
		NoTelp:         item.NoTelp,
		CreatedAt:      item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) UpdateMitra(ctx context.Context, id int32, req model.UpdateMitraRequest) (model.MitraResponse, error) {
	item, err := u.repo.UpdateMitra(ctx, entity.UpdateMitraParams{
		IDMitra:        id,
		NamaPerusahaan: req.NamaPerusahaan,
		TipePerusahaan: req.TipePerusahaan,
		Email:          req.Email,
		NoTelp:         req.NoTelp,
		Alamat:         req.Alamat,
		Kota:           req.Kota,
		KodePos:        req.KodePos,
	})
	if err != nil {
		return model.MitraResponse{}, mapMasterDataConflict(err)
	}

	return model.MitraResponse{
		ID:             item.IDMitra,
		NamaPerusahaan: item.NamaPerusahaan,
		TipePerusahaan: item.TipePerusahaan,
		Email:          item.Email,
		NoTelp:         item.NoTelp,
		CreatedAt:      item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) DeleteMitra(ctx context.Context, id int32) error {
	affected, err := u.repo.DeleteMitra(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrMasterDataNotFound
	}
	return nil
}

// BARANG
func (u *MasterDataUseCase) GetBarangByID(ctx context.Context, id int32) (model.BarangResponse, error) {
	item, err := u.repo.GetBarangByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.BarangResponse{}, ErrMasterDataNotFound
		}
		return model.BarangResponse{}, err
	}

	return model.BarangResponse{
		ID:              item.IDBarang,
		Nama:            item.NamaBarang,
		Kode:            item.Kode,
		NamaPerusahaan:  item.NamaPerusahaan,
		NamaJenisBarang: item.NamaJenisBarang,
		Satuan:          item.Satuan,
		LokasiRak:       item.LokasiRak,
		StokMinimum:     item.StokMinimum,
		CreatedAt:       item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) ListBarang(ctx context.Context, filter model.ListQueryFilter) ([]model.BarangResponse, int64, error) {
	_, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter, "created_at", true, barangSortColumns)

	items, err := u.repo.ListBarang(ctx, entity.ListBarangParams{
		SearchTerm: search,
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := u.repo.CountBarang(ctx, search)
	if err != nil {
		return nil, 0, err
	}

	res := make([]model.BarangResponse, 0, len(items))
	for _, i := range items {
		res = append(res, model.BarangResponse{
			ID:              i.IDBarang,
			Nama:            i.NamaBarang,
			Kode:            i.Kode,
			NamaPerusahaan:  i.NamaPerusahaan,
			NamaJenisBarang: i.NamaJenisBarang,
			Satuan:          i.Satuan,
			LokasiRak:       i.LokasiRak,
			StokMinimum:     i.StokMinimum,
			CreatedAt:       i.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	return res, total, nil
}

func (u *MasterDataUseCase) CreateBarang(ctx context.Context, req model.CreateBarangRequest) (model.BarangResponse, error) {
	item, err := u.repo.CreateBarang(ctx, entity.CreateBarangParams{
		NamaBarang:    req.NamaBarang,
		Kode:          req.Kode,
		IDJenisBarang: req.IDJenisBarang,
		IDMitra:       req.IDMitra,
		Satuan:        req.Satuan,
		LokasiRak:     req.LokasiRak,
		StokMinimum:   req.StokMinimum,
	})
	if err != nil {
		return model.BarangResponse{}, mapMasterDataConflict(err)
	}

	return model.BarangResponse{
		ID:          item.IDBarang,
		Nama:        item.NamaBarang,
		Kode:        item.Kode,
		Satuan:      item.Satuan,
		LokasiRak:   item.LokasiRak,
		StokMinimum: item.StokMinimum,
		CreatedAt:   item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) UpdateBarang(ctx context.Context, id int32, req model.UpdateBarangRequest) (model.BarangResponse, error) {
	item, err := u.repo.UpdateBarang(ctx, entity.UpdateBarangParams{
		IDBarang:      id,
		NamaBarang:    req.NamaBarang,
		Kode:          req.Kode,
		IDJenisBarang: req.IDJenisBarang,
		IDMitra:       req.IDMitra,
		Satuan:        req.Satuan,
		LokasiRak:     req.LokasiRak,
		StokMinimum:   req.StokMinimum,
	})
	if err != nil {
		return model.BarangResponse{}, mapMasterDataConflict(err)
	}

	return model.BarangResponse{
		ID:          item.IDBarang,
		Nama:        item.NamaBarang,
		Kode:        item.Kode,
		Satuan:      item.Satuan,
		LokasiRak:   item.LokasiRak,
		StokMinimum: item.StokMinimum,
		CreatedAt:   item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) DeleteBarang(ctx context.Context, id int32) error {
	affected, err := u.repo.DeleteBarang(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrMasterDataNotFound
	}
	return nil
}

// HAK AKSES
func (u *MasterDataUseCase) GetHakAksesByID(ctx context.Context, id int32) (model.HakAksesResponse, error) {
	item, err := u.repo.GetHakAksesByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.HakAksesResponse{}, ErrMasterDataNotFound
		}
		return model.HakAksesResponse{}, err
	}

	return model.HakAksesResponse{
		ID:        item.IDHakAkses,
		Nama:      item.NamaHalaman,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) ListHakAkses(ctx context.Context, filter model.ListQueryFilter) ([]model.HakAksesResponse, int64, error) {
	_, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter, "nama_halaman", false, hakAksesSortColumns)

	items, err := u.repo.ListHakAkses(ctx, entity.ListHakAksesParams{
		SearchTerm: search,
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := u.repo.CountHakAkses(ctx, search)
	if err != nil {
		return nil, 0, err
	}

	res := make([]model.HakAksesResponse, 0, len(items))
	for _, i := range items {
		res = append(res, model.HakAksesResponse{
			ID:        i.IDHakAkses,
			Nama:      i.NamaHalaman,
			CreatedAt: i.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return res, total, nil
}

func (u *MasterDataUseCase) CreateHakAkses(ctx context.Context, req model.CreateHakAksesRequest) (model.HakAksesResponse, error) {
	item, err := u.repo.CreateHakAkses(ctx, req.NamaHalaman)
	if err != nil {
		return model.HakAksesResponse{}, mapMasterDataConflict(err)
	}

	return model.HakAksesResponse{
		ID:        item.IDHakAkses,
		Nama:      item.NamaHalaman,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) UpdateHakAkses(ctx context.Context, id int32, req model.UpdateHakAksesRequest) (model.HakAksesResponse, error) {
	item, err := u.repo.UpdateHakAkses(ctx, entity.UpdateHakAksesParams{
		IDHakAkses:  id,
		NamaHalaman: req.NamaHalaman,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.HakAksesResponse{}, ErrMasterDataNotFound
		}
		return model.HakAksesResponse{}, mapMasterDataConflict(err)
	}

	return model.HakAksesResponse{
		ID:        item.IDHakAkses,
		Nama:      item.NamaHalaman,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) DeleteHakAkses(ctx context.Context, id int32) error {
	affected, err := u.repo.DeleteHakAkses(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrMasterDataNotFound
	}
	return nil
}

// COMPANY
func (u *MasterDataUseCase) GetCompany(ctx context.Context) (model.CompanyResponse, error) {
	item, err := u.repo.GetCompany(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.CompanyResponse{}, ErrMasterDataNotFound
		}
		return model.CompanyResponse{}, err
	}

	return model.CompanyResponse{
		ID:        item.IDCompany,
		Nama:      item.Nama,
		Alamat:    item.Alamat,
		Email:     item.Email,
		NoTelp:    item.NoTelp,
		About:     item.About,
		Logo:      item.Logo,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) GetCompanyByID(ctx context.Context, id int32) (model.CompanyResponse, error) {
	item, err := u.repo.GetCompanyByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.CompanyResponse{}, ErrMasterDataNotFound
		}
		return model.CompanyResponse{}, err
	}

	return model.CompanyResponse{
		ID:        item.IDCompany,
		Nama:      item.Nama,
		Alamat:    item.Alamat,
		Email:     item.Email,
		NoTelp:    item.NoTelp,
		About:     item.About,
		Logo:      item.Logo,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) CreateCompany(ctx context.Context, req model.CreateCompanyRequest) (model.CompanyResponse, error) {
	if _, err := u.repo.GetCompany(ctx); err == nil {
		return model.CompanyResponse{}, ErrCompanyAlreadyExists
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return model.CompanyResponse{}, fmt.Errorf("check existing company: %w", err)
	}

	item, err := u.repo.CreateCompany(ctx, entity.CreateCompanyParams{
		Nama:   req.Nama,
		Alamat: req.Alamat,
		Email:  req.Email,
		NoTelp: req.NoTelp,
		About:  req.About,
		Logo:   req.Logo,
	})
	if err != nil {
		return model.CompanyResponse{}, mapMasterDataConflict(err)
	}

	return model.CompanyResponse{
		ID:        item.IDCompany,
		Nama:      item.Nama,
		Alamat:    item.Alamat,
		Email:     item.Email,
		NoTelp:    item.NoTelp,
		About:     item.About,
		Logo:      item.Logo,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) UpdateCompany(ctx context.Context, id int32, req model.UpdateCompanyRequest) (model.CompanyResponse, error) {
	item, err := u.repo.UpdateCompany(ctx, entity.UpdateCompanyParams{
		IDCompany: id,
		Nama:      req.Nama,
		Alamat:    req.Alamat,
		Email:     req.Email,
		NoTelp:    req.NoTelp,
		About:     req.About,
		Logo:      req.Logo,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.CompanyResponse{}, ErrMasterDataNotFound
		}
		return model.CompanyResponse{}, mapMasterDataConflict(err)
	}

	return model.CompanyResponse{
		ID:        item.IDCompany,
		Nama:      item.Nama,
		Alamat:    item.Alamat,
		Email:     item.Email,
		NoTelp:    item.NoTelp,
		About:     item.About,
		Logo:      item.Logo,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) DeleteCompany(ctx context.Context, id int32) error {
	affected, err := u.repo.DeleteCompany(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrMasterDataNotFound
	}
	return nil
}

func mapMasterDataConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrMasterDataDuplicateCode
	}
	return err
}
