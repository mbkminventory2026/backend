package httpdelivery

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
)

type fakeMaterialListExcelExporter struct{ err error }

func (f fakeMaterialListExcelExporter) ExportByID(_ context.Context, _ int32) (*model.ExportedFile, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &model.ExportedFile{FileName: "MATERIAL_LIST_Test.xlsx", ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Content: []byte("PK\x03\x04xlsx")}, nil
}

func TestMaterialListExportExcel(t *testing.T) {
	tests := []struct {
		name        string
		role        string
		permissions []string
		path        string
		exportErr   error
		wantStatus  int
		wantCode    string
	}{
		{name: "authorized success", role: "ADMIN_PRODUKSI", permissions: []string{PermissionMaterialListRead}, path: "/api/v1/material-lists/7/export/excel", wantStatus: http.StatusOK},
		{name: "missing permission", role: "ADMIN_PRODUKSI", permissions: []string{PermissionWORead}, path: "/api/v1/material-lists/7/export/excel", wantStatus: http.StatusForbidden},
		{name: "client rejected", role: "CLIENT", permissions: []string{PermissionMaterialListRead}, path: "/api/v1/material-lists/7/export/excel", wantStatus: http.StatusForbidden},
		{name: "invalid id", role: "ADMIN_PRODUKSI", permissions: []string{PermissionMaterialListRead}, path: "/api/v1/material-lists/x/export/excel", wantStatus: http.StatusBadRequest},
		{name: "not found", role: "ADMIN_PRODUKSI", permissions: []string{PermissionMaterialListRead}, path: "/api/v1/material-lists/7/export/excel", exportErr: usecase.ErrMaterialListNotFound, wantStatus: http.StatusNotFound, wantCode: "material_list_not_found"},
		{name: "marker plan conflict", role: "ADMIN_PRODUKSI", permissions: []string{PermissionMaterialListRead}, path: "/api/v1/material-lists/7/export/excel", exportErr: usecase.ErrMaterialListExportMarkerPlanConflict, wantStatus: http.StatusConflict, wantCode: "marker_plan_conflict"},
		{name: "export data conflict", role: "ADMIN_PRODUKSI", permissions: []string{PermissionMaterialListRead}, path: "/api/v1/material-lists/7/export/excel", exportErr: usecase.ErrMaterialListExportDataConflict, wantStatus: http.StatusConflict, wantCode: "material_list_export_data_conflict"},
		{name: "unexpected error", role: "ADMIN_PRODUKSI", permissions: []string{PermissionMaterialListRead}, path: "/api/v1/material-lists/7/export/excel", exportErr: errors.New("database credentials leaked"), wantStatus: http.StatusInternalServerError, wantCode: "material_list_export_failed"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := materialListExportTestRouter(t, fakeMaterialListExcelExporter{err: test.exportErr}, test.role, test.permissions)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
			if recorder.Code != test.wantStatus {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			if test.wantCode != "" && !strings.Contains(recorder.Body.String(), test.wantCode) {
				t.Fatalf("response does not contain code %q: %s", test.wantCode, recorder.Body.String())
			}
			if test.name == "authorized success" {
				if got := recorder.Header().Get("Content-Type"); got != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
					t.Fatalf("content type=%q", got)
				}
				if got := recorder.Header().Get("Content-Disposition"); got != `attachment; filename="MATERIAL_LIST_Test.xlsx"` {
					t.Fatalf("content disposition=%q", got)
				}
				if !strings.HasPrefix(recorder.Body.String(), "PK\x03\x04") {
					t.Fatal("response does not start with XLSX ZIP signature")
				}
			}
			if test.name == "unexpected error" && strings.Contains(recorder.Body.String(), "database credentials leaked") {
				t.Fatalf("unsafe error leaked: %s", recorder.Body.String())
			}
		})
	}
}

func materialListExportTestRouter(t *testing.T, exporter materialListExcelExporter, role string, permissions []string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	handler, err := NewMaterialListHandler(&usecase.MaterialListUseCase{}, exporter)
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(ErrorHandlerMiddleware())
	auth := func(c *gin.Context) {
		c.Set(authorizationPayloadKey, jwt.MapClaims{"user_id": float64(7), "role_name": role, "permissions": permissions})
		c.Next()
	}
	handler.RegisterRoutes(router, auth)
	return router
}
