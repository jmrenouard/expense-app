package main

import (
    "crypto/rand"
    "database/sql"
    "encoding/csv"
    "errors"
    "fmt"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    yaml "gopkg.in/yaml.v3"
)

// Handlers groups dependencies for HTTP handlers.
type Handlers struct {
    db      *sql.DB
    datadir string
}

// NewHandlers constructs a Handlers instance.
func NewHandlers(db *sql.DB, datadir string) *Handlers {
    return &Handlers{db: db, datadir: datadir}
}

// LoginRequest represents the expected payload for authentication.
type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

// Login handles user authentication and returns a signed JWT.
func (h *Handlers) Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
        return
    }
    var user User
    err := h.db.QueryRow("SELECT id, email, password_hash, created_at FROM users WHERE email = ?", req.Email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
    if errors.Is(err, sql.ErrNoRows) {
        // Avoid leaking whether the email exists
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
        return
    } else if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    if err := checkPassword(user.PasswordHash, req.Password); err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
        return
    }
    token, err := generateJWT(user.ID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"token": token, "user": gin.H{"id": user.ID, "email": user.Email}})
}

// CreateReportRequest defines payload for creating an expense report.
type CreateReportRequest struct {
    Title string `json:"title"`
}

// CreateReport creates a new expense report with status "draft".
func (h *Handlers) CreateReport(c *gin.Context) {
    var req CreateReportRequest
    if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Title) == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
        return
    }
    // Get user ID from context
    userIDIfc, _ := c.Get(ContextUserIDKey)
    userID := userIDIfc.(int64)
    res, err := h.db.Exec("INSERT INTO expense_reports (user_id, title, status, created_at) VALUES (?, ?, ?, ?)", userID, req.Title, "draft", time.Now().UTC())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create report"})
        return
    }
    reportID, _ := res.LastInsertId()
    c.JSON(http.StatusCreated, gin.H{"id": reportID, "title": req.Title, "status": "draft"})
}

// SubmitReport sets the status of a report to "submitted". Only the report owner can submit.
func (h *Handlers) SubmitReport(c *gin.Context) {
    reportID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
        return
    }
    userIDIfc, _ := c.Get(ContextUserIDKey)
    userID := userIDIfc.(int64)
    // Ensure the report belongs to the user and is in draft
    var status string
    var ownerID int64
    err = h.db.QueryRow("SELECT id, user_id, status FROM expense_reports WHERE id = ?", reportID).Scan(&reportID, &ownerID, &status)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        }
        return
    }
    if ownerID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
        return
    }
    if status != "draft" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "only draft reports can be submitted"})
        return
    }
    if _, err := h.db.Exec("UPDATE expense_reports SET status = ? WHERE id = ?", "submitted", reportID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit report"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"id": reportID, "status": "submitted"})
}

// DeleteReport deletes a report if it belongs to the user and is still in draft.
func (h *Handlers) DeleteReport(c *gin.Context) {
    reportID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
        return
    }
    userIDIfc, _ := c.Get(ContextUserIDKey)
    userID := userIDIfc.(int64)
    // Verify conditions
    var ownerID int64
    var status string
    err = h.db.QueryRow("SELECT user_id, status FROM expense_reports WHERE id = ?", reportID).Scan(&ownerID, &status)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        }
        return
    }
    if ownerID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
        return
    }
    if status != "draft" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "only draft reports can be deleted"})
        return
    }
    if _, err := h.db.Exec("DELETE FROM expense_reports WHERE id = ?", reportID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete report"})
        return
    }
    c.Status(http.StatusNoContent)
}

// ListOwnReports returns the current user's reports.
func (h *Handlers) ListOwnReports(c *gin.Context) {
    userIDIfc, _ := c.Get(ContextUserIDKey)
    userID := userIDIfc.(int64)
    // Query reports and items joined so we can group them
    rows, err := h.db.Query(`SELECT er.id, er.title, er.status, er.created_at, ei.id, ei.description, ei.expense_date, ei.amount_ht, ei.amount_ttc, ei.vat_rate, ei.receipt_path
        FROM expense_reports er
        LEFT JOIN expense_items ei ON ei.report_id = er.id
        WHERE er.user_id = ?
        ORDER BY er.id ASC`, userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    defer rows.Close()
    type itemOut struct {
        ID          int64   `json:"id"`
        Description string  `json:"description"`
        ExpenseDate string  `json:"expense_date"`
        AmountHT    float64 `json:"amount_ht"`
        AmountTTC   float64 `json:"amount_ttc"`
        VATRate     float64 `json:"vat_rate"`
        ReceiptPath *string `json:"receipt_path,omitempty"`
    }
    type reportOut struct {
        ID        int64     `json:"id"`
        UserID    int64     `json:"user_id"`
        Title     string    `json:"title"`
        Status    string    `json:"status"`
        CreatedAt time.Time `json:"created_at"`
        Items     []itemOut `json:"items"`
    }
    reportMap := make(map[int64]*reportOut)
    for rows.Next() {
        var reportID int64
        var title string
        var status string
        var createdAt time.Time
        var itemID sql.NullInt64
        var desc sql.NullString
        var expenseDate sql.NullString
        var amtHT, amtTTC, vatRate sql.NullFloat64
        var receiptPath sql.NullString
        if err := rows.Scan(&reportID, &title, &status, &createdAt, &itemID, &desc, &expenseDate, &amtHT, &amtTTC, &vatRate, &receiptPath); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
            return
        }
        rep, ok := reportMap[reportID]
        if !ok {
            rep = &reportOut{ID: reportID, UserID: userID, Title: title, Status: status, CreatedAt: createdAt}
            reportMap[reportID] = rep
        }
        if itemID.Valid {
            itm := itemOut{
                ID:          itemID.Int64,
                Description: desc.String,
                ExpenseDate: expenseDate.String,
                AmountHT:    amtHT.Float64,
                AmountTTC:   amtTTC.Float64,
                VATRate:     vatRate.Float64,
            }
            if receiptPath.Valid {
                rp := receiptPath.String
                itm.ReceiptPath = &rp
            }
            rep.Items = append(rep.Items, itm)
        }
    }
    // Build slice
    var reports []reportOut
    for _, rep := range reportMap {
        reports = append(reports, *rep)
    }
    c.JSON(http.StatusOK, reports)
}

// AddItemRequest defines payload for adding or updating an expense item.
type AddItemRequest struct {
    Description string  `json:"description"`
    ExpenseDate string  `json:"expense_date"` // YYYY-MM-DD
    AmountHT    float64 `json:"amount_ht"`
    VATRate     float64 `json:"vat_rate"`
}

// AddItem adds an expense item to a report.
func (h *Handlers) AddItem(c *gin.Context) {
    reportID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
        return
    }
    // Verify report belongs to user
    userIDIfc, _ := c.Get(ContextUserIDKey)
    userID := userIDIfc.(int64)
    var ownerID int64
    var status string
    if err := h.db.QueryRow("SELECT user_id, status FROM expense_reports WHERE id = ?", reportID).Scan(&ownerID, &status); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        }
        return
    }
    if ownerID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
        return
    }
    if status != "draft" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "items can only be added to draft reports"})
        return
    }
    var req AddItemRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
        return
    }
    if strings.TrimSpace(req.Description) == "" || req.ExpenseDate == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "description and expense_date are required"})
        return
    }
    expDate, err := time.Parse("2006-01-02", req.ExpenseDate)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expense_date format"})
        return
    }
    amountTTC := req.AmountHT * (1 + req.VATRate)
    res, err := h.db.Exec(
        `INSERT INTO expense_items (report_id, description, expense_date, amount_ht, amount_ttc, vat_rate, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)`,
        reportID, req.Description, expDate.Format("2006-01-02"), req.AmountHT, amountTTC, req.VATRate, time.Now().UTC(),
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add item"})
        return
    }
    itemID, _ := res.LastInsertId()
    c.JSON(http.StatusCreated, gin.H{"id": itemID, "report_id": reportID, "description": req.Description, "expense_date": req.ExpenseDate, "amount_ht": req.AmountHT, "amount_ttc": amountTTC, "vat_rate": req.VATRate})
}

// UpdateItem updates an existing expense item. Only owner can update items in draft reports.
func (h *Handlers) UpdateItem(c *gin.Context) {
    itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
        return
    }
    // Get item and its report owner and status
    var reportID int64
    var ownerID int64
    var reportStatus string
    row := h.db.QueryRow(`SELECT ei.report_id, er.user_id, er.status
        FROM expense_items ei
        JOIN expense_reports er ON er.id = ei.report_id
        WHERE ei.id = ?`, itemID)
    if err := row.Scan(&reportID, &ownerID, &reportStatus); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        }
        return
    }
    userIDIfc, _ := c.Get(ContextUserIDKey)
    userID := userIDIfc.(int64)
    if ownerID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
        return
    }
    if reportStatus != "draft" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "items can only be updated in draft reports"})
        return
    }
    var req AddItemRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
        return
    }
    expDate, err := time.Parse("2006-01-02", req.ExpenseDate)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expense_date"})
        return
    }
    amountTTC := req.AmountHT * (1 + req.VATRate)
    _, err = h.db.Exec(`UPDATE expense_items SET description = ?, expense_date = ?, amount_ht = ?, amount_ttc = ?, vat_rate = ? WHERE id = ?`,
        req.Description, expDate.Format("2006-01-02"), req.AmountHT, amountTTC, req.VATRate, itemID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update item"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"id": itemID, "report_id": reportID, "description": req.Description, "expense_date": req.ExpenseDate, "amount_ht": req.AmountHT, "amount_ttc": amountTTC, "vat_rate": req.VATRate})
}

// UploadReceipt uploads or replaces the receipt attachment for an expense item.
func (h *Handlers) UploadReceipt(c *gin.Context) {
    itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
        return
    }
    // Validate file
    file, err := c.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
        return
    }
    // Check ownership and report status
    var ownerID int64
    var reportStatus string
    row := h.db.QueryRow(`SELECT er.user_id, er.status
        FROM expense_items ei
        JOIN expense_reports er ON er.id = ei.report_id
        WHERE ei.id = ?`, itemID)
    if err := row.Scan(&ownerID, &reportStatus); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        }
        return
    }
    userIDIfc, _ := c.Get(ContextUserIDKey)
    userID := userIDIfc.(int64)
    if ownerID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
        return
    }
    if reportStatus != "draft" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "receipts can only be uploaded for draft reports"})
        return
    }
    // Determine user-specific directory
    userDir := filepath.Join(h.datadir, fmt.Sprintf("%d", userID), "receipts")
    if err := os.MkdirAll(userDir, 0o755); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create receipts directory"})
        return
    }
    // Determine file extension
    ext := strings.ToLower(filepath.Ext(file.Filename))
    // Compose new filename
    newName := fmt.Sprintf("%d%s", itemID, ext)
    destPath := filepath.Join(userDir, newName)
    // Save file
    if err := c.SaveUploadedFile(file, destPath); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
        return
    }
    // Update DB
    if _, err := h.db.Exec("UPDATE expense_items SET receipt_path = ? WHERE id = ?", newName, itemID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update item"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"id": itemID, "receipt_path": newName})
}

// GetReceipt streams the receipt file if the requester has permission.
func (h *Handlers) GetReceipt(c *gin.Context) {
    itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
        return
    }
    // Retrieve item info
    var ownerID int64
    var receiptName sql.NullString
    row := h.db.QueryRow(`SELECT er.user_id, ei.receipt_path
        FROM expense_items ei
        JOIN expense_reports er ON er.id = ei.report_id
        WHERE ei.id = ?`, itemID)
    if err := row.Scan(&ownerID, &receiptName); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        }
        return
    }
    if !receiptName.Valid {
        c.JSON(http.StatusNotFound, gin.H{"error": "receipt not uploaded"})
        return
    }
    userIDIfc, _ := c.Get(ContextUserIDKey)
    userID := userIDIfc.(int64)
    // Check permission: user can always read own receipts; else must have reports:read:all
    if ownerID != userID {
        hasAll, err := UserHasPermission(h.db, userID, PermReportsReadAll)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
            return
        }
        if !hasAll {
            c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
            return
        }
    }
    // Compose full path
    filePath := filepath.Join(h.datadir, fmt.Sprintf("%d", ownerID), "receipts", receiptName.String)
    c.FileAttachment(filePath, receiptName.String)
}

// AdminListReports lists all submitted reports with user info.
func (h *Handlers) AdminListReports(c *gin.Context) {
    rows, err := h.db.Query(`SELECT er.id, er.user_id, er.title, er.status, er.created_at, u.email
        FROM expense_reports er
        JOIN users u ON u.id = er.user_id
        WHERE er.status != 'draft'`)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    defer rows.Close()
    type reportOut struct {
        ID        int64     `json:"id"`
        UserID    int64     `json:"user_id"`
        Email     string    `json:"email"`
        Title     string    `json:"title"`
        Status    string    `json:"status"`
        CreatedAt time.Time `json:"created_at"`
    }
    reports := []reportOut{}
    for rows.Next() {
        var r reportOut
        if err := rows.Scan(&r.ID, &r.UserID, &r.Title, &r.Status, &r.CreatedAt, &r.Email); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
            return
        }
        reports = append(reports, r)
    }
    c.JSON(http.StatusOK, reports)
}

// ApproveReport approves a submitted report.
func (h *Handlers) ApproveReport(c *gin.Context) {
    reportID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
        return
    }
    // Update status if submitted
    res, err := h.db.Exec(`UPDATE expense_reports SET status = 'approved' WHERE id = ? AND status = 'submitted'`, reportID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    count, _ := res.RowsAffected()
    if count == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "report not found or not in submitted state"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"id": reportID, "status": "approved"})
}

// RejectReport rejects a submitted report.
func (h *Handlers) RejectReport(c *gin.Context) {
    reportID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
        return
    }
    res, err := h.db.Exec(`UPDATE expense_reports SET status = 'rejected' WHERE id = ? AND status = 'submitted'`, reportID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    count, _ := res.RowsAffected()
    if count == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "report not found or not in submitted state"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"id": reportID, "status": "rejected"})
}

// ListUsers returns all users (id and email).
func (h *Handlers) ListUsers(c *gin.Context) {
    rows, err := h.db.Query("SELECT id, email, created_at FROM users")
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    defer rows.Close()
    var users []User
    for rows.Next() {
        var u User
        if err := rows.Scan(&u.ID, &u.Email, &u.CreatedAt); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
            return
        }
        users = append(users, u)
    }
    c.JSON(http.StatusOK, users)
}

// CreateUserRequest defines payload to create a new user.
type CreateUserRequest struct {
    Email    string  `json:"email"`
    Password string  `json:"password"`
    Groups   []int64 `json:"groups"` // list of group IDs
}

// CreateUser creates a new user and assigns groups.
func (h *Handlers) CreateUser(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
        return
    }
    if req.Email == "" || req.Password == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "email and password are required"})
        return
    }
    // Hash password
    hashed, err := hashPassword(req.Password)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
        return
    }
    tx, err := h.db.Begin()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin transaction"})
        return
    }
    defer tx.Rollback()
    res, err := tx.Exec("INSERT INTO users (email, password_hash, created_at) VALUES (?, ?, ?)", req.Email, hashed, time.Now().UTC())
    if err != nil {
        if strings.Contains(err.Error(), "UNIQUE") {
            c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
        }
        return
    }
    userID, _ := res.LastInsertId()
    // Assign groups
    for _, gid := range req.Groups {
        _, err := tx.Exec("INSERT INTO user_groups (user_id, group_id) VALUES (?, ?)", userID, gid)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign groups"})
            return
        }
    }
    if err := tx.Commit(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
        return
    }
    c.JSON(http.StatusCreated, gin.H{"id": userID, "email": req.Email})
}

// ListGroups returns all groups.
func (h *Handlers) ListGroups(c *gin.Context) {
    rows, err := h.db.Query("SELECT id, name FROM groups")
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    defer rows.Close()
    var groups []Group
    for rows.Next() {
        var g Group
        if err := rows.Scan(&g.ID, &g.Name); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
            return
        }
        groups = append(groups, g)
    }
    c.JSON(http.StatusOK, groups)
}

// CreateGroupRequest defines payload for creating a group.
type CreateGroupRequest struct {
    Name string `json:"name"`
}

// CreateGroup inserts a new group.
func (h *Handlers) CreateGroup(c *gin.Context) {
    var req CreateGroupRequest
    if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
        return
    }
    res, err := h.db.Exec("INSERT INTO groups (name) VALUES (?)", req.Name)
    if err != nil {
        if strings.Contains(err.Error(), "UNIQUE") {
            c.JSON(http.StatusConflict, gin.H{"error": "group name already exists"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create group"})
        }
        return
    }
    id, _ := res.LastInsertId()
    c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name})
}

// AssignPermissionsRequest defines payload for assigning permissions to a group.
type AssignPermissionsRequest struct {
    PermissionIDs []int64 `json:"permission_ids"`
}

// AssignPermissions assigns a list of permissions to a group.
func (h *Handlers) AssignPermissions(c *gin.Context) {
    groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
        return
    }
    var req AssignPermissionsRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
        return
    }
    tx, err := h.db.Begin()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin transaction"})
        return
    }
    defer tx.Rollback()
    for _, pid := range req.PermissionIDs {
        // Upsert assignment
        _, err := tx.Exec("INSERT OR IGNORE INTO group_permissions (group_id, permission_id) VALUES (?, ?)", groupID, pid)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign permissions"})
            return
        }
    }
    if err := tx.Commit(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"group_id": groupID, "permission_ids": req.PermissionIDs})
}

// GenerateAPIToken generates a static API token for a user.
// It stores the token in the api_tokens table creating it if necessary.
func (h *Handlers) GenerateAPIToken(c *gin.Context) {
    userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
        return
    }
    // Create table if not exists
    _, err = h.db.Exec(`CREATE TABLE IF NOT EXISTS api_tokens (
        user_id INTEGER NOT NULL,
        token TEXT NOT NULL,
        created_at DATETIME NOT NULL,
        PRIMARY KEY(user_id, token),
        FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
    )`)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create table"})
        return
    }
    // Generate random 32-byte token encoded in hex
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
        return
    }
    token := fmt.Sprintf("%x", b)
    if _, err := h.db.Exec("INSERT INTO api_tokens (user_id, token, created_at) VALUES (?, ?, ?)", userID, token, time.Now().UTC()); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store token"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"user_id": userID, "token": token})
}

// ExportCSV exports all expenses to CSV.
func (h *Handlers) ExportCSV(c *gin.Context) {
    c.Header("Content-Type", "text/csv")
    c.Header("Content-Disposition", "attachment; filename=expenses.csv")
    w := csv.NewWriter(c.Writer)
    // Write header
    w.Write([]string{"report_id", "user_id", "title", "status", "item_id", "description", "expense_date", "amount_ht", "amount_ttc", "vat_rate", "receipt_path"})
    // Query all
    rows, err := h.db.Query(`SELECT er.id, er.user_id, er.title, er.status, ei.id, ei.description, ei.expense_date, ei.amount_ht, ei.amount_ttc, ei.vat_rate, ei.receipt_path
        FROM expense_reports er
        LEFT JOIN expense_items ei ON ei.report_id = er.id`)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    defer rows.Close()
    for rows.Next() {
        var reportID, userID, itemID sql.NullInt64
        var title, status, description, receiptPath sql.NullString
        var expenseDate sql.NullString
        var amountHT, amountTTC, vatRate sql.NullFloat64
        if err := rows.Scan(&reportID, &userID, &title, &status, &itemID, &description, &expenseDate, &amountHT, &amountTTC, &vatRate, &receiptPath); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
            return
        }
        record := []string{
            fmt.Sprintf("%v", reportID.Int64),
            fmt.Sprintf("%v", userID.Int64),
            title.String,
            status.String,
            fmt.Sprintf("%v", itemID.Int64),
            description.String,
            expenseDate.String,
            fmt.Sprintf("%.2f", amountHT.Float64),
            fmt.Sprintf("%.2f", amountTTC.Float64),
            fmt.Sprintf("%.2f", vatRate.Float64),
            receiptPath.String,
        }
        w.Write(record)
    }
    w.Flush()
}

// ExportJSON exports all expenses to JSON.
func (h *Handlers) ExportJSON(c *gin.Context) {
    rows, err := h.db.Query(`SELECT er.id, er.user_id, er.title, er.status, ei.id, ei.description, ei.expense_date, ei.amount_ht, ei.amount_ttc, ei.vat_rate, ei.receipt_path
        FROM expense_reports er
        LEFT JOIN expense_items ei ON ei.report_id = er.id`)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    defer rows.Close()
    type item struct {
        ID          *int64   `json:"id,omitempty"`
        Description *string  `json:"description,omitempty"`
        ExpenseDate *string  `json:"expense_date,omitempty"`
        AmountHT    *float64 `json:"amount_ht,omitempty"`
        AmountTTC   *float64 `json:"amount_ttc,omitempty"`
        VATRate     *float64 `json:"vat_rate,omitempty"`
        ReceiptPath *string  `json:"receipt_path,omitempty"`
    }
    type report struct {
        ID     int64   `json:"id"`
        UserID int64   `json:"user_id"`
        Title  string  `json:"title"`
        Status string  `json:"status"`
        Items  []item `json:"items"`
    }
    // Group items by report
    reportMap := make(map[int64]*report)
    for rows.Next() {
        var reportID, userID, itemID sql.NullInt64
        var title, status, description, receiptPath sql.NullString
        var expenseDate sql.NullString
        var amountHT, amountTTC, vatRate sql.NullFloat64
        if err := rows.Scan(&reportID, &userID, &title, &status, &itemID, &description, &expenseDate, &amountHT, &amountTTC, &vatRate, &receiptPath); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
            return
        }
        r, ok := reportMap[reportID.Int64]
        if !ok {
            r = &report{ID: reportID.Int64, UserID: userID.Int64, Title: title.String, Status: status.String}
            reportMap[reportID.Int64] = r
        }
        if itemID.Valid {
            itm := item{}
            idVal := itemID.Int64
            itm.ID = &idVal
            descVal := description.String
            itm.Description = &descVal
            expDateVal := expenseDate.String
            itm.ExpenseDate = &expDateVal
            amtHTVal := amountHT.Float64
            itm.AmountHT = &amtHTVal
            amtTTCVal := amountTTC.Float64
            itm.AmountTTC = &amtTTCVal
            vatVal := vatRate.Float64
            itm.VATRate = &vatVal
            recVal := receiptPath.String
            if recVal != "" {
                itm.ReceiptPath = &recVal
            }
            r.Items = append(r.Items, itm)
        }
    }
    // Build slice
    var reports []report
    for _, r := range reportMap {
        reports = append(reports, *r)
    }
    c.JSON(http.StatusOK, reports)
}

// ExportYAML exports all expenses to YAML.
func (h *Handlers) ExportYAML(c *gin.Context) {
    // Use JSON data and marshal to YAML
    // Reuse ExportJSON logic
    // We'll call ExportJSON to populate w; not ideal but efficient reuse
    // Instead, call underlying logic
    // Query data
    rows, err := h.db.Query(`SELECT er.id, er.user_id, er.title, er.status, ei.id, ei.description, ei.expense_date, ei.amount_ht, ei.amount_ttc, ei.vat_rate, ei.receipt_path
        FROM expense_reports er
        LEFT JOIN expense_items ei ON ei.report_id = er.id`)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    defer rows.Close()
    type item struct {
        ID          *int64   `json:"id,omitempty" yaml:"id,omitempty"`
        Description *string  `json:"description,omitempty" yaml:"description,omitempty"`
        ExpenseDate *string  `json:"expense_date,omitempty" yaml:"expense_date,omitempty"`
        AmountHT    *float64 `json:"amount_ht,omitempty" yaml:"amount_ht,omitempty"`
        AmountTTC   *float64 `json:"amount_ttc,omitempty" yaml:"amount_ttc,omitempty"`
        VATRate     *float64 `json:"vat_rate,omitempty" yaml:"vat_rate,omitempty"`
        ReceiptPath *string  `json:"receipt_path,omitempty" yaml:"receipt_path,omitempty"`
    }
    type report struct {
        ID     int64   `json:"id" yaml:"id"`
        UserID int64   `json:"user_id" yaml:"user_id"`
        Title  string  `json:"title" yaml:"title"`
        Status string  `json:"status" yaml:"status"`
        Items  []item `json:"items" yaml:"items"`
    }
    reportMap := make(map[int64]*report)
    for rows.Next() {
        var reportID, userID, itemID sql.NullInt64
        var title, status, description, receiptPath sql.NullString
        var expenseDate sql.NullString
        var amountHT, amountTTC, vatRate sql.NullFloat64
        if err := rows.Scan(&reportID, &userID, &title, &status, &itemID, &description, &expenseDate, &amountHT, &amountTTC, &vatRate, &receiptPath); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
            return
        }
        r, ok := reportMap[reportID.Int64]
        if !ok {
            r = &report{ID: reportID.Int64, UserID: userID.Int64, Title: title.String, Status: status.String}
            reportMap[reportID.Int64] = r
        }
        if itemID.Valid {
            itm := item{}
            idVal := itemID.Int64
            itm.ID = &idVal
            descVal := description.String
            itm.Description = &descVal
            expDateVal := expenseDate.String
            itm.ExpenseDate = &expDateVal
            amtHTVal := amountHT.Float64
            itm.AmountHT = &amtHTVal
            amtTTCVal := amountTTC.Float64
            itm.AmountTTC = &amtTTCVal
            vatVal := vatRate.Float64
            itm.VATRate = &vatVal
            recVal := receiptPath.String
            if recVal != "" {
                itm.ReceiptPath = &recVal
            }
            r.Items = append(r.Items, itm)
        }
    }
    var reports []report
    for _, r := range reportMap {
        reports = append(reports, *r)
    }
    // Marshal to YAML
    out, err := yaml.Marshal(reports)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal YAML"})
        return
    }
    c.Header("Content-Type", "application/x-yaml")
    c.Header("Content-Disposition", "attachment; filename=expenses.yaml")
    c.Data(http.StatusOK, "application/x-yaml", out)
}
