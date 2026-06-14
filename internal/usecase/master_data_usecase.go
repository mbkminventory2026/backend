package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

var (
	ErrMasterDataNotFound      = errors.New("master data not found")
	ErrMasterDataConflict      = errors.New("master data already exists")
	ErrMasterDataDuplicateCode = errors.New("master data code already exists")
	ErrMasterDataInUse         = errors.New("master data is in use")

	departemenSortColumns  = buildSortWhitelist("created_at", "id_departemen", "nama_departemen")
	jenisBarangSortColumns = buildSortWhitelist("created_at", "id_jenis_barang", "kode", "nama_jenis_barang")
	mitraSortColumns       = buildSortWhitelist("created_at", "id_mitra", "nama_perusahaan", "email", "no_telp", "tipe_perusahaan")
	barangSortColumns      = buildSortWhitelist("created_at", "id_barang", "kode", "nama_barang", "nama_jenis_barang", "nama_perusahaan")
	hakAksesSortColumns    = buildSortWhitelist("created_at", "id_hak_akses", "nama_halaman", "kode_permission", "domain_permission", "aksi_permission")
	warnaSortColumns       = buildSortWhitelist("created_at", "id_warna", "nama_warna")
	sizeSortColumns        = buildSortWhitelist("created_at", "id_size", "nama_size")
)

type MasterDataUseCase struct {
	repo     entity.Querier
	auditLog *AuditLogUseCase
}

func NewMasterDataUseCase(repo entity.Querier, auditLog *AuditLogUseCase) (*MasterDataUseCase, error) {
	if repo == nil {
		return nil, errors.New("repository is required")
	}
	return &MasterDataUseCase{repo: repo, auditLog: auditLog}, nil
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

	result := model.DepartemenResponse{
		ID:        item.IDDepartemen,
		Nama:      item.NamaDepartemen,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordDepartemenCreateAudit(ctx, result)

	return result, nil
}

func (u *MasterDataUseCase) UpdateDepartemen(ctx context.Context, id int32, req model.UpdateDepartemenRequest) (model.DepartemenResponse, error) {
	beforeItem, err := u.GetDepartemenByID(ctx, id)
	if err != nil {
		return model.DepartemenResponse{}, err
	}

	item, err := u.repo.UpdateDepartemen(ctx, entity.UpdateDepartemenParams{
		IDDepartemen:   id,
		NamaDepartemen: req.NamaDepartemen,
	})
	if err != nil {
		return model.DepartemenResponse{}, mapMasterDataConflict(err)
	}

	result := model.DepartemenResponse{
		ID:        item.IDDepartemen,
		Nama:      item.NamaDepartemen,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordDepartemenUpdateAudit(ctx, result, buildDepartemenAuditSnapshot(beforeItem), buildDepartemenAuditSnapshot(result))

	return result, nil
}

func (u *MasterDataUseCase) DeleteDepartemen(ctx context.Context, id int32) error {
	existing, err := u.GetDepartemenByID(ctx, id)
	if err != nil {
		return err
	}

	affected, err := u.repo.DeleteDepartemen(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrMasterDataNotFound
	}

	u.recordDepartemenDeleteAudit(ctx, existing)

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

	result := model.JenisBarangResponse{
		ID:        item.IDJenisBarang,
		Nama:      item.NamaJenisBarang,
		Kode:      item.Kode,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordJenisBarangCreateAudit(ctx, result)

	return result, nil
}

func (u *MasterDataUseCase) UpdateJenisBarang(ctx context.Context, id int32, req model.UpdateJenisBarangRequest) (model.JenisBarangResponse, error) {
	beforeItem, err := u.GetJenisBarangByID(ctx, id)
	if err != nil {
		return model.JenisBarangResponse{}, err
	}

	item, err := u.repo.UpdateJenisBarang(ctx, entity.UpdateJenisBarangParams{
		IDJenisBarang:   id,
		NamaJenisBarang: req.NamaJenisBarang,
		Kode:            req.Kode,
	})
	if err != nil {
		return model.JenisBarangResponse{}, mapMasterDataConflict(err)
	}

	result := model.JenisBarangResponse{
		ID:        item.IDJenisBarang,
		Nama:      item.NamaJenisBarang,
		Kode:      item.Kode,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordJenisBarangUpdateAudit(ctx, result, buildJenisBarangAuditSnapshot(beforeItem), buildJenisBarangAuditSnapshot(result))

	return result, nil
}

func (u *MasterDataUseCase) DeleteJenisBarang(ctx context.Context, id int32) error {
	existing, err := u.GetJenisBarangByID(ctx, id)
	if err != nil {
		return err
	}

	affected, err := u.repo.DeleteJenisBarang(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrMasterDataNotFound
	}

	u.recordJenisBarangDeleteAudit(ctx, existing)

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
		Alamat:         item.Alamat,
		Kota:           item.Kota,
		KodePos:        item.KodePos,
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
			Alamat:         i.Alamat,
			Kota:           i.Kota,
			KodePos:        i.KodePos,
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

	result := model.MitraResponse{
		ID:             item.IDMitra,
		NamaPerusahaan: item.NamaPerusahaan,
		TipePerusahaan: item.TipePerusahaan,
		Email:          item.Email,
		NoTelp:         item.NoTelp,
		Alamat:         item.Alamat,
		Kota:           item.Kota,
		KodePos:        item.KodePos,
		CreatedAt:      item.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordMitraCreateAudit(ctx, result)

	return result, nil
}

func (u *MasterDataUseCase) UpdateMitra(ctx context.Context, id int32, req model.UpdateMitraRequest) (model.MitraResponse, error) {
	beforeItem, err := u.GetMitraByID(ctx, id)
	if err != nil {
		return model.MitraResponse{}, err
	}

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

	result := model.MitraResponse{
		ID:             item.IDMitra,
		NamaPerusahaan: item.NamaPerusahaan,
		TipePerusahaan: item.TipePerusahaan,
		Email:          item.Email,
		NoTelp:         item.NoTelp,
		Alamat:         item.Alamat,
		Kota:           item.Kota,
		KodePos:        item.KodePos,
		CreatedAt:      item.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordMitraUpdateAudit(ctx, result, buildMitraAuditSnapshot(beforeItem), buildMitraAuditSnapshot(result))

	return result, nil
}

func (u *MasterDataUseCase) DeleteMitra(ctx context.Context, id int32) error {
	existing, err := u.GetMitraByID(ctx, id)
	if err != nil {
		return err
	}

	affected, err := u.repo.DeleteMitra(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrMasterDataNotFound
	}

	u.recordMitraDeleteAudit(ctx, existing)

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

	result, err := u.GetBarangByID(ctx, item.IDBarang)
	if err != nil {
		return model.BarangResponse{}, err
	}

	u.recordBarangCreateAudit(ctx, result)

	return result, nil
}

func (u *MasterDataUseCase) UpdateBarang(ctx context.Context, id int32, req model.UpdateBarangRequest) (model.BarangResponse, error) {
	beforeItem, err := u.GetBarangByID(ctx, id)
	if err != nil {
		return model.BarangResponse{}, err
	}

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

	result, err := u.GetBarangByID(ctx, item.IDBarang)
	if err != nil {
		return model.BarangResponse{}, err
	}

	u.recordBarangUpdateAudit(ctx, result, buildBarangAuditSnapshot(beforeItem), buildBarangAuditSnapshot(result))

	return result, nil
}

func (u *MasterDataUseCase) DeleteBarang(ctx context.Context, id int32) error {
	existing, err := u.GetBarangByID(ctx, id)
	if err != nil {
		return err
	}

	affected, err := u.repo.DeleteBarang(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrMasterDataNotFound
	}

	u.recordBarangDeleteAudit(ctx, existing)

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
		ID:               item.IDHakAkses,
		KodePermission:   item.KodePermission,
		Nama:             item.NamaHalaman,
		Deskripsi:        item.Deskripsi,
		DomainPermission: item.DomainPermission,
		AksiPermission:   item.AksiPermission,
		CreatedAt:        item.CreatedAt.Time.Format(time.RFC3339),
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
			ID:               i.IDHakAkses,
			KodePermission:   i.KodePermission,
			Nama:             i.NamaHalaman,
			Deskripsi:        i.Deskripsi,
			DomainPermission: i.DomainPermission,
			AksiPermission:   i.AksiPermission,
			CreatedAt:        i.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return res, total, nil
}

func (u *MasterDataUseCase) CreateHakAkses(ctx context.Context, req model.CreateHakAksesRequest) (model.HakAksesResponse, error) {
	item, err := u.repo.CreateHakAkses(ctx, entity.CreateHakAksesParams{
		KodePermission:   req.KodePermission,
		NamaHalaman:      req.NamaHalaman,
		Deskripsi:        req.Deskripsi,
		DomainPermission: req.DomainPermission,
		AksiPermission:   req.AksiPermission,
	})
	if err != nil {
		return model.HakAksesResponse{}, mapMasterDataConflict(err)
	}

	result := model.HakAksesResponse{
		ID:               item.IDHakAkses,
		KodePermission:   item.KodePermission,
		Nama:             item.NamaHalaman,
		Deskripsi:        item.Deskripsi,
		DomainPermission: item.DomainPermission,
		AksiPermission:   item.AksiPermission,
		CreatedAt:        item.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordHakAksesCreateAudit(ctx, result)

	return result, nil
}

func (u *MasterDataUseCase) UpdateHakAkses(ctx context.Context, id int32, req model.UpdateHakAksesRequest) (model.HakAksesResponse, error) {
	beforeItem, err := u.GetHakAksesByID(ctx, id)
	if err != nil {
		return model.HakAksesResponse{}, err
	}

	item, err := u.repo.UpdateHakAkses(ctx, entity.UpdateHakAksesParams{
		IDHakAkses:       id,
		KodePermission:   req.KodePermission,
		NamaHalaman:      req.NamaHalaman,
		Deskripsi:        req.Deskripsi,
		DomainPermission: req.DomainPermission,
		AksiPermission:   req.AksiPermission,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.HakAksesResponse{}, ErrMasterDataNotFound
		}
		return model.HakAksesResponse{}, mapMasterDataConflict(err)
	}

	result := model.HakAksesResponse{
		ID:               item.IDHakAkses,
		KodePermission:   item.KodePermission,
		Nama:             item.NamaHalaman,
		Deskripsi:        item.Deskripsi,
		DomainPermission: item.DomainPermission,
		AksiPermission:   item.AksiPermission,
		CreatedAt:        item.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordHakAksesUpdateAudit(ctx, result, buildHakAksesAuditSnapshot(beforeItem), buildHakAksesAuditSnapshot(result))

	return result, nil
}

func (u *MasterDataUseCase) DeleteHakAkses(ctx context.Context, id int32) error {
	existing, err := u.GetHakAksesByID(ctx, id)
	if err != nil {
		return err
	}

	affected, err := u.repo.DeleteHakAkses(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrMasterDataNotFound
	}

	u.recordHakAksesDeleteAudit(ctx, existing)

	return nil
}

func mapMasterDataConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrMasterDataDuplicateCode
	}
	return err
}

func mapMasterDataDeleteConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return ErrMasterDataInUse
	}
	return err
}

// WARNA
func (u *MasterDataUseCase) GetWarnaByID(ctx context.Context, id int32) (model.WarnaResponse, error) {
	item, err := u.repo.GetWarnaByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.WarnaResponse{}, ErrMasterDataNotFound
		}
		return model.WarnaResponse{}, err
	}

	return model.WarnaResponse{
		ID:        item.IDWarna,
		NamaWarna: item.NamaWarna,
		KodeHex:   pgTextToPtrString(item.KodeHex),
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) ListWarna(ctx context.Context, filter model.ListQueryFilter) ([]model.WarnaResponse, int64, error) {
	_, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter, "nama_warna", false, warnaSortColumns)

	items, err := u.repo.ListWarna(ctx, entity.ListWarnaParams{
		SearchTerm: search,
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := u.repo.CountWarna(ctx, search)
	if err != nil {
		return nil, 0, err
	}

	res := make([]model.WarnaResponse, 0, len(items))
	for _, i := range items {
		res = append(res, model.WarnaResponse{
			ID:        i.IDWarna,
			NamaWarna: i.NamaWarna,
			KodeHex:   pgTextToPtrString(i.KodeHex),
			CreatedAt: i.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	return res, total, nil
}

func (u *MasterDataUseCase) CreateWarna(ctx context.Context, req model.CreateWarnaRequest) (model.WarnaResponse, error) {
	item, err := u.repo.CreateWarna(ctx, entity.CreateWarnaParams{
		NamaWarna: req.NamaWarna,
		KodeHex:   ptrStringToPgText(req.KodeHex),
	})
	if err != nil {
		return model.WarnaResponse{}, mapMasterDataConflict(err)
	}

	result := model.WarnaResponse{
		ID:        item.IDWarna,
		NamaWarna: item.NamaWarna,
		KodeHex:   pgTextToPtrString(item.KodeHex),
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordWarnaCreateAudit(ctx, result)

	return result, nil
}

func (u *MasterDataUseCase) UpdateWarna(ctx context.Context, id int32, req model.UpdateWarnaRequest) (model.WarnaResponse, error) {
	beforeItem, err := u.GetWarnaByID(ctx, id)
	if err != nil {
		return model.WarnaResponse{}, err
	}

	item, err := u.repo.UpdateWarna(ctx, entity.UpdateWarnaParams{
		IDWarna:   id,
		NamaWarna: req.NamaWarna,
		KodeHex:   ptrStringToPgText(req.KodeHex),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.WarnaResponse{}, ErrMasterDataNotFound
		}
		return model.WarnaResponse{}, mapMasterDataConflict(err)
	}

	result := model.WarnaResponse{
		ID:        item.IDWarna,
		NamaWarna: item.NamaWarna,
		KodeHex:   pgTextToPtrString(item.KodeHex),
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordWarnaUpdateAudit(ctx, result, buildWarnaAuditSnapshot(beforeItem), buildWarnaAuditSnapshot(result))

	return result, nil
}

func (u *MasterDataUseCase) DeleteWarna(ctx context.Context, id int32) error {
	existing, err := u.GetWarnaByID(ctx, id)
	if err != nil {
		return err
	}

	affected, err := u.repo.DeleteWarna(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrMasterDataNotFound
	}

	u.recordWarnaDeleteAudit(ctx, existing)

	return nil
}

// SIZE
func (u *MasterDataUseCase) GetSizeByID(ctx context.Context, id int32) (model.SizeResponse, error) {
	item, err := u.repo.GetSizeByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.SizeResponse{}, ErrMasterDataNotFound
		}
		return model.SizeResponse{}, err
	}

	return model.SizeResponse{
		ID:        item.IDSize,
		NamaSize:  item.NamaSize,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *MasterDataUseCase) ListSizes(ctx context.Context, filter model.ListQueryFilter) ([]model.SizeResponse, int64, error) {
	_, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter, "nama_size", false, sizeSortColumns)

	items, err := u.repo.ListSizes(ctx, entity.ListSizesParams{
		SearchTerm: search,
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := u.repo.CountSizes(ctx, search)
	if err != nil {
		return nil, 0, err
	}

	res := make([]model.SizeResponse, 0, len(items))
	for _, i := range items {
		res = append(res, model.SizeResponse{
			ID:        i.IDSize,
			NamaSize:  i.NamaSize,
			CreatedAt: i.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	return res, total, nil
}

func (u *MasterDataUseCase) CreateSize(ctx context.Context, req model.CreateSizeRequest) (model.SizeResponse, error) {
	item, err := u.repo.CreateSize(ctx, req.NamaSize)
	if err != nil {
		return model.SizeResponse{}, mapMasterDataConflict(err)
	}

	result := model.SizeResponse{
		ID:        item.IDSize,
		NamaSize:  item.NamaSize,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordSizeCreateAudit(ctx, result)

	return result, nil
}

func (u *MasterDataUseCase) UpdateSize(ctx context.Context, id int32, req model.UpdateSizeRequest) (model.SizeResponse, error) {
	beforeItem, err := u.GetSizeByID(ctx, id)
	if err != nil {
		return model.SizeResponse{}, err
	}

	item, err := u.repo.UpdateSize(ctx, entity.UpdateSizeParams{
		IDSize:   id,
		NamaSize: req.NamaSize,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.SizeResponse{}, ErrMasterDataNotFound
		}
		return model.SizeResponse{}, mapMasterDataConflict(err)
	}

	result := model.SizeResponse{
		ID:        item.IDSize,
		NamaSize:  item.NamaSize,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordSizeUpdateAudit(ctx, result, buildSizeAuditSnapshot(beforeItem), buildSizeAuditSnapshot(result))

	return result, nil
}

func (u *MasterDataUseCase) DeleteSize(ctx context.Context, id int32) error {
	existing, err := u.GetSizeByID(ctx, id)
	if err != nil {
		return err
	}

	affected, err := u.repo.DeleteSize(ctx, id)
	if err != nil {
		return mapMasterDataDeleteConflict(err)
	}
	if affected == 0 {
		return ErrMasterDataNotFound
	}

	u.recordSizeDeleteAudit(ctx, existing)

	return nil
}

func ptrStringToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func pgTextToPtrString(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func (u *MasterDataUseCase) recordHakAksesCreateAudit(ctx context.Context, item model.HakAksesResponse) {
	if u.auditLog == nil {
		return
	}

	auditCtx, ok := GetAuditLogContext(ctx)
	if !ok {
		return
	}

	if err := u.auditLog.Record(ctx, model.AuditLogRecordRequest{
		ActorUserID: auditCtx.ActorUserID,
		ActorRole:   auditCtx.ActorRole,
		Action:      "CREATE",
		Module:      "role-management",
		EntityType:  "hak_akses",
		EntityID:    fmt.Sprintf("%d", item.ID),
		EntityLabel: item.KodePermission,
		Method:      auditCtx.Method,
		Route:       auditCtx.Route,
		AfterData:   buildHakAksesAuditSnapshot(item),
	}); err != nil {
		slog.Error("failed to record permission create audit log", slog.String("error", err.Error()))
	}
}

func (u *MasterDataUseCase) recordHakAksesUpdateAudit(ctx context.Context, item model.HakAksesResponse, beforeSnapshot, afterSnapshot map[string]any) {
	if u.auditLog == nil {
		return
	}

	auditCtx, ok := GetAuditLogContext(ctx)
	if !ok {
		return
	}

	if err := u.auditLog.Record(ctx, model.AuditLogRecordRequest{
		ActorUserID:   auditCtx.ActorUserID,
		ActorRole:     auditCtx.ActorRole,
		Action:        "UPDATE",
		Module:        "role-management",
		EntityType:    "hak_akses",
		EntityID:      fmt.Sprintf("%d", item.ID),
		EntityLabel:   item.KodePermission,
		Method:        auditCtx.Method,
		Route:         auditCtx.Route,
		BeforeData:    beforeSnapshot,
		AfterData:     afterSnapshot,
		ChangedFields: buildChangedFieldsFromSnapshots(beforeSnapshot, afterSnapshot),
	}); err != nil {
		slog.Error("failed to record permission update audit log", slog.String("error", err.Error()))
	}
}

func (u *MasterDataUseCase) recordHakAksesDeleteAudit(ctx context.Context, item model.HakAksesResponse) {
	if u.auditLog == nil {
		return
	}

	auditCtx, ok := GetAuditLogContext(ctx)
	if !ok {
		return
	}

	if err := u.auditLog.Record(ctx, model.AuditLogRecordRequest{
		ActorUserID: auditCtx.ActorUserID,
		ActorRole:   auditCtx.ActorRole,
		Action:      "DELETE",
		Module:      "role-management",
		EntityType:  "hak_akses",
		EntityID:    fmt.Sprintf("%d", item.ID),
		EntityLabel: item.KodePermission,
		Method:      auditCtx.Method,
		Route:       auditCtx.Route,
		BeforeData:  buildHakAksesAuditSnapshot(item),
	}); err != nil {
		slog.Error("failed to record permission delete audit log", slog.String("error", err.Error()))
	}
}

func buildHakAksesAuditSnapshot(item model.HakAksesResponse) map[string]any {
	return map[string]any{
		"id_hak_akses":      item.ID,
		"kode_permission":   item.KodePermission,
		"nama_halaman":      item.Nama,
		"deskripsi":         item.Deskripsi,
		"domain_permission": item.DomainPermission,
		"aksi_permission":   item.AksiPermission,
	}
}

func (u *MasterDataUseCase) recordDepartemenCreateAudit(ctx context.Context, item model.DepartemenResponse) {
	u.recordMasterDataAudit(ctx, "CREATE", "departemen", fmt.Sprintf("%d", item.ID), item.Nama, nil, buildDepartemenAuditSnapshot(item))
}

func (u *MasterDataUseCase) recordDepartemenUpdateAudit(ctx context.Context, item model.DepartemenResponse, beforeSnapshot, afterSnapshot map[string]any) {
	u.recordMasterDataAudit(ctx, "UPDATE", "departemen", fmt.Sprintf("%d", item.ID), item.Nama, beforeSnapshot, afterSnapshot)
}

func (u *MasterDataUseCase) recordDepartemenDeleteAudit(ctx context.Context, item model.DepartemenResponse) {
	u.recordMasterDataAudit(ctx, "DELETE", "departemen", fmt.Sprintf("%d", item.ID), item.Nama, buildDepartemenAuditSnapshot(item), nil)
}

func buildDepartemenAuditSnapshot(item model.DepartemenResponse) map[string]any {
	return map[string]any{
		"id_departemen":   item.ID,
		"nama_departemen": item.Nama,
	}
}

func (u *MasterDataUseCase) recordJenisBarangCreateAudit(ctx context.Context, item model.JenisBarangResponse) {
	u.recordMasterDataAudit(ctx, "CREATE", "jenis_barang", fmt.Sprintf("%d", item.ID), item.Nama, nil, buildJenisBarangAuditSnapshot(item))
}

func (u *MasterDataUseCase) recordJenisBarangUpdateAudit(ctx context.Context, item model.JenisBarangResponse, beforeSnapshot, afterSnapshot map[string]any) {
	u.recordMasterDataAudit(ctx, "UPDATE", "jenis_barang", fmt.Sprintf("%d", item.ID), item.Nama, beforeSnapshot, afterSnapshot)
}

func (u *MasterDataUseCase) recordJenisBarangDeleteAudit(ctx context.Context, item model.JenisBarangResponse) {
	u.recordMasterDataAudit(ctx, "DELETE", "jenis_barang", fmt.Sprintf("%d", item.ID), item.Nama, buildJenisBarangAuditSnapshot(item), nil)
}

func buildJenisBarangAuditSnapshot(item model.JenisBarangResponse) map[string]any {
	return map[string]any{
		"id_jenis_barang":   item.ID,
		"nama_jenis_barang": item.Nama,
		"kode":              item.Kode,
	}
}

func (u *MasterDataUseCase) recordMitraCreateAudit(ctx context.Context, item model.MitraResponse) {
	u.recordMasterDataAudit(ctx, "CREATE", "mitra", fmt.Sprintf("%d", item.ID), item.NamaPerusahaan, nil, buildMitraAuditSnapshot(item))
}

func (u *MasterDataUseCase) recordMitraUpdateAudit(ctx context.Context, item model.MitraResponse, beforeSnapshot, afterSnapshot map[string]any) {
	u.recordMasterDataAudit(ctx, "UPDATE", "mitra", fmt.Sprintf("%d", item.ID), item.NamaPerusahaan, beforeSnapshot, afterSnapshot)
}

func (u *MasterDataUseCase) recordMitraDeleteAudit(ctx context.Context, item model.MitraResponse) {
	u.recordMasterDataAudit(ctx, "DELETE", "mitra", fmt.Sprintf("%d", item.ID), item.NamaPerusahaan, buildMitraAuditSnapshot(item), nil)
}

func buildMitraAuditSnapshot(item model.MitraResponse) map[string]any {
	return map[string]any{
		"id_mitra":        item.ID,
		"nama_perusahaan": item.NamaPerusahaan,
		"tipe_perusahaan": item.TipePerusahaan,
		"email":           item.Email,
		"no_telp":         item.NoTelp,
		"alamat":          item.Alamat,
		"kota":            item.Kota,
		"kode_pos":        item.KodePos,
	}
}

func (u *MasterDataUseCase) recordBarangCreateAudit(ctx context.Context, item model.BarangResponse) {
	u.recordMasterDataAudit(ctx, "CREATE", "barang", fmt.Sprintf("%d", item.ID), item.Nama, nil, buildBarangAuditSnapshot(item))
}

func (u *MasterDataUseCase) recordBarangUpdateAudit(ctx context.Context, item model.BarangResponse, beforeSnapshot, afterSnapshot map[string]any) {
	u.recordMasterDataAudit(ctx, "UPDATE", "barang", fmt.Sprintf("%d", item.ID), item.Nama, beforeSnapshot, afterSnapshot)
}

func (u *MasterDataUseCase) recordBarangDeleteAudit(ctx context.Context, item model.BarangResponse) {
	u.recordMasterDataAudit(ctx, "DELETE", "barang", fmt.Sprintf("%d", item.ID), item.Nama, buildBarangAuditSnapshot(item), nil)
}

func buildBarangAuditSnapshot(item model.BarangResponse) map[string]any {
	return map[string]any{
		"id_barang":         item.ID,
		"nama_barang":       item.Nama,
		"kode":              item.Kode,
		"nama_perusahaan":   item.NamaPerusahaan,
		"nama_jenis_barang": item.NamaJenisBarang,
		"satuan":            item.Satuan,
		"lokasi_rak":        item.LokasiRak,
		"stok_minimum":      item.StokMinimum,
	}
}

func (u *MasterDataUseCase) recordWarnaCreateAudit(ctx context.Context, item model.WarnaResponse) {
	u.recordMasterDataAudit(ctx, "CREATE", "warna", fmt.Sprintf("%d", item.ID), item.NamaWarna, nil, buildWarnaAuditSnapshot(item))
}

func (u *MasterDataUseCase) recordWarnaUpdateAudit(ctx context.Context, item model.WarnaResponse, beforeSnapshot, afterSnapshot map[string]any) {
	u.recordMasterDataAudit(ctx, "UPDATE", "warna", fmt.Sprintf("%d", item.ID), item.NamaWarna, beforeSnapshot, afterSnapshot)
}

func (u *MasterDataUseCase) recordWarnaDeleteAudit(ctx context.Context, item model.WarnaResponse) {
	u.recordMasterDataAudit(ctx, "DELETE", "warna", fmt.Sprintf("%d", item.ID), item.NamaWarna, buildWarnaAuditSnapshot(item), nil)
}

func buildWarnaAuditSnapshot(item model.WarnaResponse) map[string]any {
	return map[string]any{
		"id_warna":   item.ID,
		"nama_warna": item.NamaWarna,
		"kode_hex":   item.KodeHex,
	}
}

func (u *MasterDataUseCase) recordSizeCreateAudit(ctx context.Context, item model.SizeResponse) {
	u.recordMasterDataAudit(ctx, "CREATE", "size", fmt.Sprintf("%d", item.ID), item.NamaSize, nil, buildSizeAuditSnapshot(item))
}

func (u *MasterDataUseCase) recordSizeUpdateAudit(ctx context.Context, item model.SizeResponse, beforeSnapshot, afterSnapshot map[string]any) {
	u.recordMasterDataAudit(ctx, "UPDATE", "size", fmt.Sprintf("%d", item.ID), item.NamaSize, beforeSnapshot, afterSnapshot)
}

func (u *MasterDataUseCase) recordSizeDeleteAudit(ctx context.Context, item model.SizeResponse) {
	u.recordMasterDataAudit(ctx, "DELETE", "size", fmt.Sprintf("%d", item.ID), item.NamaSize, buildSizeAuditSnapshot(item), nil)
}

func buildSizeAuditSnapshot(item model.SizeResponse) map[string]any {
	return map[string]any{
		"id_size":   item.ID,
		"nama_size": item.NamaSize,
	}
}

func (u *MasterDataUseCase) recordMasterDataAudit(ctx context.Context, action, entityType, entityID, entityLabel string, beforeSnapshot, afterSnapshot map[string]any) {
	if u.auditLog == nil {
		return
	}

	auditCtx, ok := GetAuditLogContext(ctx)
	if !ok {
		return
	}

	recordRequest := model.AuditLogRecordRequest{
		ActorUserID: auditCtx.ActorUserID,
		ActorRole:   auditCtx.ActorRole,
		Action:      action,
		Module:      "master-data",
		EntityType:  entityType,
		EntityID:    entityID,
		EntityLabel: entityLabel,
		Method:      auditCtx.Method,
		Route:       auditCtx.Route,
		BeforeData:  beforeSnapshot,
		AfterData:   afterSnapshot,
	}

	if action == "UPDATE" {
		recordRequest.ChangedFields = buildChangedFieldsFromSnapshots(beforeSnapshot, afterSnapshot)
	}

	if err := u.auditLog.Record(ctx, recordRequest); err != nil {
		slog.Error("failed to record master data audit log", slog.String("entity_type", entityType), slog.String("action", action), slog.String("error", err.Error()))
	}
}
