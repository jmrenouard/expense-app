package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

func setupTestServer(t *testing.T) (*gin.Engine, *sql.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	datadir := t.TempDir()
	dbPath := filepath.Join(datadir, "expense.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	if err := InitDB(db); err != nil {
		if strings.Contains(err.Error(), "requires cgo") {
			t.Skipf("SQLite integration tests require CGO_ENABLED=1: %v", err)
		}
		require.NoError(t, err)
	}
	return SetupRouter(db, datadir), db
}

func doJSONRequest(t *testing.T, r *gin.Engine, method, path, token string, payload interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func loginAndGetToken(t *testing.T, r *gin.Engine, email, password string) string {
	t.Helper()
	rec := doJSONRequest(t, r, http.MethodPost, "/api/auth/login", "", map[string]string{
		"email":    email,
		"password": password,
	})
	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Token)
	return resp.Token
}

func TestSeededGroupsAdminLoginAndAPIKeyGuard(t *testing.T) {
	r, db := setupTestServer(t)
	adminToken := loginAndGetToken(t, r, "admin@example.com", "admin")

	// Verify seeded French group names match models.go InitDB
	rec := doJSONRequest(t, r, http.MethodGet, "/api/admin/groups", adminToken, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var groups []Group
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &groups))
	names := make([]string, 0, len(groups))
	for _, g := range groups {
		names = append(names, g.Name)
	}
	assert.ElementsMatch(t, []string{"Administrateurs", "Validateurs", "Utilisateurs"}, names)

	// Verify GenerateAPIToken persists token while X-API-Key stays rejected with 401
	tokRec := doJSONRequest(t, r, http.MethodPost, "/api/admin/users/1/token", adminToken, nil)
	require.Equal(t, http.StatusOK, tokRec.Code)
	var tokOut struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(tokRec.Body.Bytes(), &tokOut))
	assert.Len(t, tokOut.Token, 64)

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM api_tokens WHERE user_id = 1").Scan(&count))
	assert.Equal(t, 1, count)

	apiKeyReq := httptest.NewRequest(http.MethodGet, "/api/reports", nil)
	apiKeyReq.Header.Set("X-API-Key", tokOut.Token)
	apiKeyRec := httptest.NewRecorder()
	r.ServeHTTP(apiKeyRec, apiKeyReq)
	assert.Equal(t, http.StatusUnauthorized, apiKeyRec.Code)
}

func TestTutorialExpenseLifecycleReceiptsRBACAndExports(t *testing.T) {
	r, _ := setupTestServer(t)
	adminToken := loginAndGetToken(t, r, "admin@example.com", "admin")

	// Resolve Utilisateurs group ID
	groupsRec := doJSONRequest(t, r, http.MethodGet, "/api/admin/groups", adminToken, nil)
	require.Equal(t, http.StatusOK, groupsRec.Code)
	var groups []Group
	require.NoError(t, json.Unmarshal(groupsRec.Body.Bytes(), &groups))
	var userGroupID int64
	for _, g := range groups {
		if g.Name == "Utilisateurs" {
			userGroupID = g.ID
		}
	}
	require.NotZero(t, userGroupID)

	// Create standard user and authenticate
	createUserRec := doJSONRequest(t, r, http.MethodPost, "/api/admin/users", adminToken, map[string]interface{}{
		"email":    "employee@example.com",
		"password": "userpass123",
		"groups":   []int64{userGroupID},
	})
	require.Equal(t, http.StatusCreated, createUserRec.Code)
	userToken := loginAndGetToken(t, r, "employee@example.com", "userpass123")

	// Standard user is forbidden from admin approval and exports
	forbidRec := doJSONRequest(t, r, http.MethodGet, "/api/admin/export/json", userToken, nil)
	assert.Equal(t, http.StatusForbidden, forbidRec.Code)

	// Create draft report
	repRec := doJSONRequest(t, r, http.MethodPost, "/api/reports", userToken, map[string]string{
		"title": "August Client Travel",
	})
	require.Equal(t, http.StatusCreated, repRec.Code)
	var repOut struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(repRec.Body.Bytes(), &repOut))
	assert.Equal(t, "draft", repOut.Status)

	// Add itemized expense and verify VAT calculation: 100 * (1 + 0.20) = 120
	itemRec := doJSONRequest(t, r, http.MethodPost, fmt.Sprintf("/api/reports/%d/items", repOut.ID), userToken, map[string]interface{}{
		"description":  "Train ticket",
		"expense_date": "2026-08-12",
		"amount_ht":    100.0,
		"vat_rate":     0.20,
	})
	require.Equal(t, http.StatusCreated, itemRec.Code)
	var itemOut struct {
		ID        int64   `json:"id"`
		AmountTTC float64 `json:"amount_ttc"`
	}
	require.NoError(t, json.Unmarshal(itemRec.Body.Bytes(), &itemOut))
	assert.InDelta(t, 120.0, itemOut.AmountTTC, 0.001)

	// Upload multipart receipt and verify download content
	var mpBuf bytes.Buffer
	mw := multipart.NewWriter(&mpBuf)
	fw, err := mw.CreateFormFile("file", "receipt.txt")
	require.NoError(t, err)
	_, err = io.WriteString(fw, "receipt-payload-bytes")
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	upReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/items/%d/receipt", itemOut.ID), &mpBuf)
	upReq.Header.Set("Content-Type", mw.FormDataContentType())
	upReq.Header.Set("Authorization", "Bearer "+userToken)
	upRec := httptest.NewRecorder()
	r.ServeHTTP(upRec, upReq)
	require.Equal(t, http.StatusOK, upRec.Code)

	getReceiptRec := doJSONRequest(t, r, http.MethodGet, fmt.Sprintf("/api/items/%d/receipt", itemOut.ID), userToken, nil)
	require.Equal(t, http.StatusOK, getReceiptRec.Code)
	assert.Equal(t, "receipt-payload-bytes", getReceiptRec.Body.String())

	// Submit report and verify non-draft delete guard
	subRec := doJSONRequest(t, r, http.MethodPost, fmt.Sprintf("/api/reports/%d/submit", repOut.ID), userToken, nil)
	require.Equal(t, http.StatusOK, subRec.Code)
	delSubmittedRec := doJSONRequest(t, r, http.MethodDelete, fmt.Sprintf("/api/reports/%d", repOut.ID), userToken, nil)
	assert.Equal(t, http.StatusBadRequest, delSubmittedRec.Code)

	// Standard user cannot self-approve; admin approves
	selfApproveRec := doJSONRequest(t, r, http.MethodPost, fmt.Sprintf("/api/admin/reports/%d/approve", repOut.ID), userToken, nil)
	assert.Equal(t, http.StatusForbidden, selfApproveRec.Code)

	approveRec := doJSONRequest(t, r, http.MethodPost, fmt.Sprintf("/api/admin/reports/%d/approve", repOut.ID), adminToken, nil)
	require.Equal(t, http.StatusOK, approveRec.Code)

	// Verify draft deletion works cleanly
	draft2Rec := doJSONRequest(t, r, http.MethodPost, "/api/reports", userToken, map[string]string{"title": "Temp Draft"})
	require.Equal(t, http.StatusCreated, draft2Rec.Code)
	var draft2 struct {
		ID int64 `json:"id"`
	}
	require.NoError(t, json.Unmarshal(draft2Rec.Body.Bytes(), &draft2))
	delDraftRec := doJSONRequest(t, r, http.MethodDelete, fmt.Sprintf("/api/reports/%d", draft2.ID), userToken, nil)
	assert.Equal(t, http.StatusNoContent, delDraftRec.Code)

	// Verify CSV, JSON, and YAML admin exports match approved report
	csvRec := doJSONRequest(t, r, http.MethodGet, "/api/admin/export/csv", adminToken, nil)
	require.Equal(t, http.StatusOK, csvRec.Code)
	rows, err := csv.NewReader(strings.NewReader(csvRec.Body.String())).ReadAll()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), 2)
	assert.Equal(t, "report_id", rows[0][0])

	jsonRec := doJSONRequest(t, r, http.MethodGet, "/api/admin/export/json", adminToken, nil)
	require.Equal(t, http.StatusOK, jsonRec.Code)
	var exportedJSON []map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonRec.Body.Bytes(), &exportedJSON))
	require.Len(t, exportedJSON, 1)
	assert.Equal(t, "approved", exportedJSON[0]["status"])

	yamlRec := doJSONRequest(t, r, http.MethodGet, "/api/admin/export/yaml", adminToken, nil)
	require.Equal(t, http.StatusOK, yamlRec.Code)
	var exportedYAML []map[string]interface{}
	require.NoError(t, yaml.Unmarshal(yamlRec.Body.Bytes(), &exportedYAML))
	require.Len(t, exportedYAML, 1)
	assert.Equal(t, "August Client Travel", exportedYAML[0]["title"])
}
