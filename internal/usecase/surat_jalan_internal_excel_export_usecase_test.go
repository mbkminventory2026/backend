package usecase

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/pkg/exporter/excel"
)

func TestSuratJalanInternalTemplateLayoutAndDataMapping(t *testing.T) {
	renderer, err := excel.NewRenderer(filepath.Join("..", "..", "templates", "exports"))
	if err != nil {
		t.Fatal(err)
	}
	book, err := renderer.OpenTemplate(suratJalanInternalExportTemplateName)
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	sheet := book.GetSheetName(0)
	detail := &model.SuratJalanInternalDetailResponse{ID: 7, NoDokumen: "SJI-007", Buyer: "PT Tujuan", Model: "STYLE-01", CreatedAt: "2026-08-09T00:00:00Z", WOShells: []model.SuratJalanInternalShellRow{{No: 1, Qty: 3, Deskripsi: "Kain utama"}, {No: 2, Qty: 5, Deskripsi: "Aksesori"}}}
	workOrder := &model.WorkOrderDetailResponse{PONumber: "PO-123", POClientItemStyle: "STYLE-01"}
	layout, err := prepareSuratJalanInternalExportLayout(book, sheet, len(detail.WOShells))
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSuratJalanInternalExportHeader(book, sheet, detail, workOrder, model.ProfilPerusahaanResponse{}); err != nil {
		t.Fatal(err)
	}
	if err := writeSuratJalanInternalExportItems(book, sheet, layout, detail.WOShells); err != nil {
		t.Fatal(err)
	}
	for cell, want := range map[string]string{"I1": "SURAT JALAN", "I2": "Delivery Order", "A11": "NO", "B11": "QTY", "C11": "UNIT", "D11": "NAMA BARANG", "C4": "SJI-007", "C7": "PO-123", "C8": "STYLE-01", "I5": "PT Tujuan", "A12": "1", "B13": "5", "D12": "Kain utama", "A21": "TOTAL", "B21": "8"} {
		got, err := book.GetCellValue(sheet, cell)
		if err != nil || got != want {
			t.Fatalf("%s = %q, want %q (err=%v)", cell, got, want, err)
		}
	}
	if got, _ := book.GetCellValue(sheet, "C12"); got != "" {
		t.Fatalf("unit must remain blank without authoritative data: %q", got)
	}
	assertSuratJalanGreen(t, book, sheet, 27)
	for _, cell := range []string{"A11", "D14", "L14"} {
		styleID, err := book.GetCellStyle(sheet, cell)
		if err != nil {
			t.Fatal(err)
		}
		style, err := book.GetStyle(styleID)
		if err != nil {
			t.Fatal(err)
		}
		if cell == "A11" && (len(style.Fill.Color) == 0 || style.Fill.Color[0] != "00AF50") {
			t.Fatalf("%s fill is not corporate green: %#v", cell, style.Fill.Color)
		}
		for _, border := range style.Border {
			if border.Color != "00AF50" {
				t.Fatalf("%s border is not corporate green: %q", cell, border.Color)
			}
		}
	}
	dynamicBook, err := renderer.OpenTemplate(suratJalanInternalExportTemplateName)
	if err != nil {
		t.Fatal(err)
	}
	defer dynamicBook.Close()
	if _, err := prepareSuratJalanInternalExportLayout(dynamicBook, dynamicBook.GetSheetName(0), 10); err != nil {
		t.Fatal(err)
	}
	assertSuratJalanGreen(t, dynamicBook, dynamicBook.GetSheetName(0), 28)
	for _, cell := range []string{"D21", "L21"} {
		styleID, err := dynamicBook.GetCellStyle(sheet, cell)
		if err != nil {
			t.Fatal(err)
		}
		style, err := dynamicBook.GetStyle(styleID)
		if err != nil || len(style.Border) == 0 {
			t.Fatalf("dynamic row %s lost its grid style: %v", cell, err)
		}
	}
	if output := os.Getenv("PERMATATEX_SURAT_JALAN_SAMPLE"); output != "" {
		if err := book.SaveAs(output); err != nil {
			t.Fatal(err)
		}
	}
	logo := image.NewRGBA(image.Rect(0, 0, 2, 2))
	logo.Set(0, 0, color.RGBA{R: 0, G: 175, B: 80, A: 255})
	var logoBytes bytes.Buffer
	if err := png.Encode(&logoBytes, logo); err != nil {
		t.Fatal(err)
	}
	if err := insertSuratJalanInternalCompanyLogo(book, sheet, "data:image/png;base64,"+base64.StdEncoding.EncodeToString(logoBytes.Bytes())); err != nil {
		t.Fatal(err)
	}
	pictures, err := book.GetPictures(sheet, "A1")
	if err != nil || len(pictures) != 1 {
		t.Fatalf("runtime logo pictures = %d, err=%v", len(pictures), err)
	}
}

func assertSuratJalanGreen(t *testing.T, workbook *excelize.File, sheet string, lastRow int) {
	t.Helper()
	for row := 1; row <= lastRow; row++ {
		for column := 1; column <= 12; column++ {
			cell, err := excelize.CoordinatesToCellName(column, row)
			if err != nil {
				t.Fatal(err)
			}
			styleID, err := workbook.GetCellStyle(sheet, cell)
			if err != nil {
				t.Fatal(err)
			}
			style, err := workbook.GetStyle(styleID)
			if err != nil {
				t.Fatal(err)
			}
			if style.Font != nil && style.Font.Color != "" && style.Font.Color != "FFFFFF" && style.Font.Color != "00AF50" {
				t.Fatalf("%s has non-corporate direct font color %q", cell, style.Font.Color)
			}
			for _, fill := range style.Fill.Color {
				if fill != "" && fill != "00AF50" {
					t.Fatalf("%s has non-corporate fill color %q", cell, fill)
				}
			}
			for _, border := range style.Border {
				if border.Style != 0 && border.Color != "00AF50" {
					t.Fatalf("%s %s border color = %q", cell, border.Type, border.Color)
				}
			}
		}
	}
}
