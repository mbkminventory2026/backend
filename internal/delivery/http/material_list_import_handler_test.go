package httpdelivery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/xuri/excelize/v2"
	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
)

type importHandlerExporter struct{}

func (importHandlerExporter) ExportByID(context.Context, int32) (*model.ExportedFile, error) {
	return nil, nil
}

type importHandlerTemplate struct{}

func (importHandlerTemplate) Generate(context.Context, int32) (*model.ExportedFile, error) {
	return &model.ExportedFile{FileName: "MATERIAL_LIST_IMPORT.xlsx", ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Content: []byte("xlsx")}, nil
}

type countedTemplate struct {
	err      error
	calls    int
	fileName string
}

func (u *countedTemplate) Generate(context.Context, int32) (*model.ExportedFile, error) {
	u.calls++
	if u.err != nil {
		return nil, u.err
	}
	return &model.ExportedFile{FileName: u.fileName, ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Content: validTemplateBytes()}, nil
}

type importHandlerUseCase struct{ err error }

func (u importHandlerUseCase) Import(_ context.Context, _ int32, r io.Reader) (*usecase.MaterialListExcelImportResult, error) {
	_, _ = io.ReadAll(r)
	if u.err != nil {
		return nil, u.err
	}
	return &usecase.MaterialListExcelImportResult{CreatedCount: 2}, nil
}

type countedImporter struct {
	err   error
	calls int
}

func (u *countedImporter) Import(_ context.Context, _ int32, r io.Reader) (*usecase.MaterialListExcelImportResult, error) {
	u.calls++
	_, _ = io.ReadAll(r)
	if u.err != nil {
		return nil, u.err
	}
	return &usecase.MaterialListExcelImportResult{CreatedCount: 2}, nil
}

func validTemplateBytes() []byte {
	f := excelize.NewFile()
	defer f.Close()
	_ = f.SetCellValue("Sheet1", "A1", "ok")
	var b bytes.Buffer
	_ = f.Write(&b)
	return b.Bytes()
}
func multipartRequest(t *testing.T, files ...struct {
	name    string
	content []byte
}) *http.Request {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for _, file := range files {
		p, e := w.CreateFormFile("file", file.name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = p.Write(file.content); e != nil {
			t.Fatal(e)
		}
	}
	if e := w.Close(); e != nil {
		t.Fatal(e)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/material-lists/7/import/excel", &b)
	r.Header.Set("Content-Type", w.FormDataContentType())
	return r
}
func importRouter(t *testing.T, imp materialListExcelImporter, tmpl materialListExcelImportTemplater, role string, permissions []string) *gin.Engine {
	t.Helper()
	h, e := NewMaterialListHandler(&usecase.MaterialListUseCase{}, importHandlerExporter{}, imp, tmpl)
	if e != nil {
		t.Fatal(e)
	}
	r := gin.New()
	r.Use(ErrorHandlerMiddleware())
	auth := func(c *gin.Context) {
		c.Set(authorizationPayloadKey, jwt.MapClaims{"user_id": float64(7), "role_name": role, "permissions": permissions})
		c.Next()
	}
	h.RegisterRoutes(r, auth)
	return r
}

func TestMaterialListImportHandlerSuccessAndFileValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, err := NewMaterialListHandler(&usecase.MaterialListUseCase{}, importHandlerExporter{}, importHandlerUseCase{}, importHandlerTemplate{})
	if err != nil {
		t.Fatal(err)
	}
	request := func(name string, content []byte) *http.Request {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		p, e := w.CreateFormFile("file", name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = p.Write(content); e != nil {
			t.Fatal(e)
		}
		if e = w.Close(); e != nil {
			t.Fatal(e)
		}
		r := httptest.NewRequest(http.MethodPost, "/api/v1/material-lists/7/import/excel", &b)
		r.Header.Set("Content-Type", w.FormDataContentType())
		return r
	}
	for _, tc := range []struct {
		name, file string
		want       int
	}{{"valid", "input.xlsx", http.StatusCreated}, {"extension", "input.csv", http.StatusBadRequest}} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := gin.New()
			r.Use(ErrorHandlerMiddleware())
			r.POST("/api/v1/material-lists/:id/import/excel", h.ImportExcel)
			r.ServeHTTP(w, request(tc.file, []byte("content")))
			if w.Code != tc.want {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if tc.want == http.StatusCreated {
				var v struct {
					Data map[string]int `json:"data"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil || v.Data["id_material_list"] != 7 || v.Data["created_count"] != 2 {
					t.Fatalf("response=%s err=%v", w.Body.String(), err)
				}
			}
		})
	}
}

func TestMaterialListImportHandlerConflictMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	conflict := &usecase.MaterialListExcelImportValidationError{Errors: []usecase.MaterialListExcelImportError{{Row: 2, Column: "ITEM", Code: "duplicate_in_file", Message: "duplicate"}}, ErrorCount: 1}
	h, err := NewMaterialListHandler(&usecase.MaterialListUseCase{}, importHandlerExporter{}, importHandlerUseCase{err: conflict}, importHandlerTemplate{})
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	p, _ := mw.CreateFormFile("file", "input.xlsx")
	_, _ = p.Write([]byte("x"))
	_ = mw.Close()
	w := httptest.NewRecorder()
	r := gin.New()
	r.Use(ErrorHandlerMiddleware())
	r.POST("/api/v1/material-lists/:id/import/excel", h.ImportExcel)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/material-lists/7/import/excel", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestMaterialListImportErrorStatusDeterministic(t *testing.T) {
	cases := []struct {
		name  string
		codes []string
		want  int
	}{
		{"ordinary", []string{"invalid_reference"}, 400}, {"mixed", []string{"duplicate_in_file", "invalid_reference"}, 400},
		{"duplicate file", []string{"duplicate_in_file"}, 409}, {"duplicate existing", []string{"duplicate_existing_item"}, 409}, {"duplicates", []string{"duplicate_existing_item", "duplicate_in_file"}, 409},
		{"locked", []string{"material_list_locked"}, 409}, {"not found", []string{"material_list_not_found"}, 404}, {"too large", []string{"file_too_large"}, 413},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := &usecase.MaterialListExcelImportValidationError{}
			for _, code := range tc.codes {
				v.Errors = append(v.Errors, usecase.MaterialListExcelImportError{Row: 2, Column: "ITEM", Code: code, Message: "safe"})
				v.ErrorCount++
			}
			_, got := importErrorStatus(v)
			if got != tc.want {
				t.Fatalf("got %d", got)
			}
		})
	}
}

func TestMaterialListImportAndTemplateAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, role  string
		permissions []string
		path        string
		want        int
		template    bool
	}{
		{"import update", "ADMIN_PRODUKSI", []string{PermissionMaterialListUpdate}, "/api/v1/material-lists/7/import/excel", 201, false},
		{"import read only", "ADMIN_PRODUKSI", []string{PermissionMaterialListRead}, "/api/v1/material-lists/7/import/excel", 403, false},
		{"import client", "CLIENT", []string{PermissionMaterialListUpdate}, "/api/v1/material-lists/7/import/excel", 403, false},
		{"template update", "ADMIN_PRODUKSI", []string{PermissionMaterialListUpdate}, "/api/v1/material-lists/7/import/excel/template", 200, true},
		{"template read only", "ADMIN_PRODUKSI", []string{PermissionMaterialListRead}, "/api/v1/material-lists/7/import/excel/template", 403, true},
		{"template client", "CLIENT", []string{PermissionMaterialListUpdate}, "/api/v1/material-lists/7/import/excel/template", 403, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			imp := &countedImporter{}
			tmpl := &countedTemplate{fileName: "MATERIAL_LIST_IMPORT_Test.xlsx"}
			r := importRouter(t, imp, tmpl, tc.role, tc.permissions)
			w := httptest.NewRecorder()
			var req *http.Request
			if tc.template {
				req = httptest.NewRequest(http.MethodGet, tc.path, nil)
			} else {
				req = multipartRequest(t, struct {
					name    string
					content []byte
				}{"x.xlsx", []byte("x")})
			}
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if tc.want == 403 && (imp.calls != 0 || tmpl.calls != 0) {
				t.Fatal("authorization invoked usecase")
			}
			if tc.template && tc.want == 200 {
				if w.Header().Get("Content-Type") != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" || !strings.Contains(w.Header().Get("Content-Disposition"), "MATERIAL_LIST_IMPORT_Test.xlsx") {
					t.Fatalf("headers=%v", w.Header())
				}
				if _, e := excelize.OpenReader(bytes.NewReader(w.Body.Bytes())); e != nil {
					t.Fatal(e)
				}
			}
		})
	}
}

func TestMaterialListTemplateStatuses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name string
		err  error
		want int
		code string
	}{
		{"not found", &usecase.MaterialListExcelImportValidationError{Errors: []usecase.MaterialListExcelImportError{{Code: "material_list_not_found"}}, ErrorCount: 1}, 404, "material_list_not_found"},
		{"locked", &usecase.MaterialListExcelImportValidationError{Errors: []usecase.MaterialListExcelImportError{{Code: "material_list_locked"}}, ErrorCount: 1}, 409, "material_list_locked"},
		{"safe internal", errors.New("database password leaked"), 500, "material_list_import_failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := &countedTemplate{err: tc.err}
			r := importRouter(t, &countedImporter{}, tmpl, "ADMIN_PRODUKSI", []string{PermissionMaterialListUpdate})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/material-lists/7/import/excel/template", nil))
			if w.Code != tc.want || !strings.Contains(w.Body.String(), tc.code) || strings.Contains(w.Body.String(), "password leaked") {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestMaterialListImportHTTPErrorMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validation := func(codes ...string) error {
		v := &usecase.MaterialListExcelImportValidationError{}
		for _, c := range codes {
			v.Errors = append(v.Errors, usecase.MaterialListExcelImportError{Row: 2, Column: "ITEM", Code: c, Message: "safe"})
			v.ErrorCount++
		}
		return v
	}
	cases := []struct {
		name string
		err  error
		want int
		code string
	}{
		{"malformed", validation("invalid_workbook"), 400, "invalid_workbook"}, {"unsupported", validation("unsupported_template_version"), 400, "unsupported_template_version"}, {"mismatch", validation("target_material_list_mismatch"), 400, "target_material_list_mismatch"}, {"reference", validation("invalid_reference"), 400, "invalid_workbook"}, {"row", validation("invalid_enum"), 400, "invalid_workbook"}, {"mixed forward", validation("duplicate_in_file", "invalid_reference"), 400, "invalid_workbook"}, {"mixed reverse", validation("invalid_reference", "duplicate_in_file"), 400, "invalid_workbook"}, {"limit", validation("row_limit_exceeded"), 400, "invalid_workbook"},
		{"duplicate file", validation("duplicate_in_file"), 409, "duplicate_in_file"}, {"duplicate existing", validation("duplicate_existing_item"), 409, "duplicate_in_file"}, {"duplicates", validation("duplicate_existing_item", "duplicate_in_file"), 409, "duplicate_in_file"}, {"locked", validation("material_list_locked"), 409, "material_list_locked"}, {"core size", validation("file_too_large"), 413, "file_too_large"}, {"not found", validation("material_list_not_found"), 404, "material_list_not_found"}, {"internal", errors.New("sql credentials"), 500, "material_list_import_failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			imp := &countedImporter{err: tc.err}
			r := importRouter(t, imp, &countedTemplate{}, "ADMIN_PRODUKSI", []string{PermissionMaterialListUpdate})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, multipartRequest(t, struct {
				name    string
				content []byte
			}{"x.xlsx", []byte("x")}))
			if w.Code != tc.want || !strings.Contains(w.Body.String(), tc.code) || strings.Contains(w.Body.String(), "credentials") {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestMaterialListImportMultipartGuards(t *testing.T) {
	gin.SetMode(gin.TestMode)
	imp := &countedImporter{}
	r := importRouter(t, imp, &countedTemplate{}, "ADMIN_PRODUKSI", []string{PermissionMaterialListUpdate})
	cases := []struct {
		name string
		req  func() *http.Request
		want int
	}{
		{"missing", func() *http.Request {
			return httptest.NewRequest(http.MethodPost, "/api/v1/material-lists/7/import/excel", nil)
		}, 400},
		{"csv", func() *http.Request {
			return multipartRequest(t, struct {
				name    string
				content []byte
			}{"x.csv", []byte("x")})
		}, 400}, {"xls", func() *http.Request {
			return multipartRequest(t, struct {
				name    string
				content []byte
			}{"x.xls", []byte("x")})
		}, 400}, {"xlsm", func() *http.Request {
			return multipartRequest(t, struct {
				name    string
				content []byte
			}{"x.xlsm", []byte("x")})
		}, 400},
		{"multiple", func() *http.Request {
			return multipartRequest(t, struct {
				name    string
				content []byte
			}{"a.xlsx", []byte("x")}, struct {
				name    string
				content []byte
			}{"b.xlsx", []byte("x")})
		}, 400},
		{"body ceiling", func() *http.Request {
			return multipartRequest(t, struct {
				name    string
				content []byte
			}{"x.xlsx", make([]byte, materialListImportHTTPMaxBytes)})
		}, 413},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, tc.req())
			if w.Code != tc.want {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
	if imp.calls != 0 {
		t.Fatalf("guard failures called importer %d times", imp.calls)
	}
}
