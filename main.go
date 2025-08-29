package main

import (
    "database/sql"
    "fmt"
    "log"
    "os"
    "path/filepath"

    "github.com/gin-gonic/gin"
    _ "github.com/mattn/go-sqlite3"
)

func main() {
    // Determine data directory. Defaults to ./data
    datadir := os.Getenv("DATADIR")
    if datadir == "" {
        datadir = "./data"
    }
    // Ensure datadir exists
    if err := os.MkdirAll(datadir, 0o755); err != nil {
        log.Fatalf("failed to create data dir: %v", err)
    }
    // Connect to SQLite database stored in datadir
    dbPath := filepath.Join(datadir, "expense.db")
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        log.Fatalf("failed to open database: %v", err)
    }
    defer db.Close()
    // Initialize database schema and seed data
    if err := InitDB(db); err != nil {
        log.Fatalf("failed to init database: %v", err)
    }
    // Load translations
    if err := LoadTranslations("i18n"); err != nil {
        log.Fatalf("failed to load translations: %v", err)
    }
    // Create handlers
    handlers := NewHandlers(db, datadir)
    r := gin.Default()
    // Public routes
    r.POST("/api/auth/login", handlers.Login)
    // Protected routes with authentication
    api := r.Group("/api")
    api.Use(AuthMiddleware(db))
    {
        // Reports
        api.POST("/reports", RequirePermission(db, PermReportsCreate), handlers.CreateReport)
        api.POST("/reports/:id/submit", RequirePermission(db, PermReportsUpdateOwn), handlers.SubmitReport)
        api.DELETE("/reports/:id", RequirePermission(db, PermReportsUpdateOwn), handlers.DeleteReport)
        api.GET("/reports", RequirePermission(db, PermReportsReadOwn), handlers.ListOwnReports)
        // Items
        api.POST("/reports/:id/items", RequirePermission(db, PermReportsCreate), handlers.AddItem)
        api.PUT("/items/:id", RequirePermission(db, PermReportsUpdateOwn), handlers.UpdateItem)
        api.POST("/items/:id/receipt", RequirePermission(db, PermReportsUpdateOwn), handlers.UploadReceipt)
        api.GET("/items/:id/receipt", RequirePermission(db, PermReportsReadOwn), handlers.GetReceipt)
        // Admin sub routes
        admin := api.Group("/admin")
        {
            admin.GET("/reports", RequirePermission(db, PermReportsReadAll), handlers.AdminListReports)
            admin.POST("/reports/:id/approve", RequirePermission(db, PermReportsApprove), handlers.ApproveReport)
            admin.POST("/reports/:id/reject", RequirePermission(db, PermReportsReject), handlers.RejectReport)
            admin.GET("/users", RequirePermission(db, PermUsersRead), handlers.ListUsers)
            admin.POST("/users", RequirePermission(db, PermUsersCreate), handlers.CreateUser)
            admin.GET("/groups", RequirePermission(db, PermGroupsRead), handlers.ListGroups)
            admin.POST("/groups", RequirePermission(db, PermGroupsCreate), handlers.CreateGroup)
            admin.POST("/groups/:id/permissions", RequirePermission(db, PermPermissionsAssign), handlers.AssignPermissions)
            admin.POST("/users/:id/token", RequirePermission(db, PermTokensCreate), handlers.GenerateAPIToken)
            admin.GET("/export/csv", RequirePermission(db, PermReportsExportAll), handlers.ExportCSV)
            admin.GET("/export/json", RequirePermission(db, PermReportsExportAll), handlers.ExportJSON)
            admin.GET("/export/yaml", RequirePermission(db, PermReportsExportAll), handlers.ExportYAML)
        }
    }
    // Determine server port
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    addr := fmt.Sprintf(":%s", port)
    log.Printf("Server listening on %s", addr)
    if err := r.Run(addr); err != nil {
        log.Fatalf("failed to run server: %v", err)
    }
}
