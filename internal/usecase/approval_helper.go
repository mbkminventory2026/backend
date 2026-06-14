package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"permatatex-inventory/internal/entity"
)

func initializeApprovalWorkflow(ctx context.Context, qtx entity.Querier, tableName string, docID int32, creatorUserID int32) error {
	header, err := qtx.CreateApprovalHeader(ctx, entity.CreateApprovalHeaderParams{
		NamaTabelDokumen: tableName,
		IDDokumen:        docID,
		StatusGlobal:     "pending",
	})
	if err != nil {
		return err
	}

	fallbackID := creatorUserID
	if fallbackID <= 0 {
		mgrUsers, err := qtx.GetUsersByRoleName(ctx, "MANAGER")
		if err == nil && len(mgrUsers) > 0 {
			fallbackID = mgrUsers[0].IDUser
		} else {
			keuUsers, err := qtx.GetUsersByRoleName(ctx, "ADMIN_KEUANGAN")
			if err == nil && len(keuUsers) > 0 {
				fallbackID = keuUsers[0].IDUser
			} else {
				fallbackID = 1 // Last resort fallback
			}
		}
	}

	switch tableName {
	case "PR_INTERNAL":
		// Step 1: PEMBUAT (Creator) - marked as done immediately
		now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
		_, err = qtx.CreateApprovalDetail(ctx, entity.CreateApprovalDetailParams{
			IDOtoritas:   header,
			IDUser:       creatorUserID,
			TipePeran:    "PEMBUAT",
			IsActionDone: true,
			WaktuAksi:    now,
			Catatan:      "Dokumen PR dibuat",
		})
		if err != nil {
			return err
		}

		// Step 2: PENGECEK (Admin Produksi)
		produksiID := getAssigneeForRole(ctx, qtx, "ADMIN_PRODUKSI", fallbackID)
		_, err = qtx.CreateApprovalDetail(ctx, entity.CreateApprovalDetailParams{
			IDOtoritas:   header,
			IDUser:       produksiID,
			TipePeran:    "PENGECEK",
			IsActionDone: false,
		})
		if err != nil {
			return err
		}

		// Step 3: PENYETUJU (Manager)
		managerID := getAssigneeForRole(ctx, qtx, "MANAGER", fallbackID)
		_, err = qtx.CreateApprovalDetail(ctx, entity.CreateApprovalDetailParams{
			IDOtoritas:   header,
			IDUser:       managerID,
			TipePeran:    "PENYETUJU",
			IsActionDone: false,
		})
		if err != nil {
			return err
		}

		// Step 4: RELEASE (Admin Keuangan)
		keuanganID := getAssigneeForRole(ctx, qtx, "ADMIN_KEUANGAN", fallbackID)
		_, err = qtx.CreateApprovalDetail(ctx, entity.CreateApprovalDetailParams{
			IDOtoritas:   header,
			IDUser:       keuanganID,
			TipePeran:    "RELEASE",
			IsActionDone: false,
		})
		if err != nil {
			return err
		}

	case "WORK_ORDER", "PO_INTERNAL", "MARKER_PLAN", "TIMELINE_PRODUKSI", "PACKING_LIST", "SPREADING_CUTTING_PLAN", "DATA_APPROVE_CUTTING_PLAN":
		// Alur standar 2-langkah: Pembuat -> Manager
		now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
		_, err = qtx.CreateApprovalDetail(ctx, entity.CreateApprovalDetailParams{
			IDOtoritas:   header,
			IDUser:       creatorUserID,
			TipePeran:    "PEMBUAT",
			IsActionDone: true,
			WaktuAksi:    now,
			Catatan:      fmt.Sprintf("Dokumen %s dibuat", tableName),
		})
		if err != nil {
			return err
		}

		managerID := getAssigneeForRole(ctx, qtx, "MANAGER", fallbackID)
		_, err = qtx.CreateApprovalDetail(ctx, entity.CreateApprovalDetailParams{
			IDOtoritas:   header,
			IDUser:       managerID,
			TipePeran:    "PENYETUJU",
			IsActionDone: false,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func getAssigneeForRole(ctx context.Context, qtx entity.Querier, roleName string, fallbackID int32) int32 {
	// 1. Try target role
	users, err := qtx.GetUsersByRoleName(ctx, roleName)
	if err == nil && len(users) > 0 {
		return users[0].IDUser
	}

	// 2. Fallback to MANAGER
	if roleName != "MANAGER" {
		mgrUsers, err := qtx.GetUsersByRoleName(ctx, "MANAGER")
		if err == nil && len(mgrUsers) > 0 {
			return mgrUsers[0].IDUser
		}
	}

	// 3. Fallback to ADMIN_KEUANGAN
	if roleName != "ADMIN_KEUANGAN" {
		keuUsers, err := qtx.GetUsersByRoleName(ctx, "ADMIN_KEUANGAN")
		if err == nil && len(keuUsers) > 0 {
			return keuUsers[0].IDUser
		}
	}

	// 4. Ultimate fallback to creatorUserID
	return fallbackID
}
