package main

import (
    "database/sql"
    "errors"
    "fmt"
    "os"
    "time"
)

// User represents a system user. PasswordHash stores the bcrypt hashed password.
type User struct {
    ID           int64     `db:"id" json:"id"`
    Email        string    `db:"email" json:"email"`
    PasswordHash string    `db:"password_hash" json:"-"`
    CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// Group represents a collection of permissions.
type Group struct {
    ID   int64  `db:"id" json:"id"`
    Name string `db:"name" json:"name"`
}

// Permission represents a single atomic permission.
type Permission struct {
    ID     int64  `db:"id" json:"id"`
    Action string `db:"action" json:"action"`
}

// ExpenseReport represents a report containing multiple expense items.
type ExpenseReport struct {
    ID        int64     `db:"id" json:"id"`
    UserID    int64     `db:"user_id" json:"user_id"`
    Title     string    `db:"title" json:"title"`
    Status    string    `db:"status" json:"status"`
    CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// ExpenseItem represents a single expense within a report.
type ExpenseItem struct {
    ID          int64     `db:"id" json:"id"`
    ReportID    int64     `db:"report_id" json:"report_id"`
    Description string    `db:"description" json:"description"`
    ExpenseDate time.Time `db:"expense_date" json:"expense_date"`
    AmountHT    float64   `db:"amount_ht" json:"amount_ht"`
    AmountTTC   float64   `db:"amount_ttc" json:"amount_ttc"`
    VATRate     float64   `db:"vat_rate" json:"vat_rate"`
    ReceiptPath string    `db:"receipt_path" json:"receipt_path"`
    CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// InitDB creates all required tables and seeds default data.
// It is idempotent and can be called multiple times.
func InitDB(db *sql.DB) error {
    // Enable foreign keys in SQLite.
    if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
        return fmt.Errorf("enable foreign keys: %w", err)
    }
    // Create USERS table
    usersTable := `CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        email TEXT NOT NULL UNIQUE,
        password_hash TEXT NOT NULL,
        created_at DATETIME NOT NULL
    )`;
    if _, err := db.Exec(usersTable); err != nil {
        return fmt.Errorf("create users: %w", err)
    }
    // Create GROUPS table
    groupsTable := `CREATE TABLE IF NOT EXISTS groups (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL UNIQUE
    )`;
    if _, err := db.Exec(groupsTable); err != nil {
        return fmt.Errorf("create groups: %w", err)
    }
    // Create PERMISSIONS table
    permsTable := `CREATE TABLE IF NOT EXISTS permissions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        action TEXT NOT NULL UNIQUE
    )`;
    if _, err := db.Exec(permsTable); err != nil {
        return fmt.Errorf("create permissions: %w", err)
    }
    // Create USER_GROUPS table
    userGroupsTable := `CREATE TABLE IF NOT EXISTS user_groups (
        user_id INTEGER NOT NULL,
        group_id INTEGER NOT NULL,
        PRIMARY KEY(user_id, group_id),
        FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
        FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
    )`;
    if _, err := db.Exec(userGroupsTable); err != nil {
        return fmt.Errorf("create user_groups: %w", err)
    }
    // Create GROUP_PERMISSIONS table
    groupPermsTable := `CREATE TABLE IF NOT EXISTS group_permissions (
        group_id INTEGER NOT NULL,
        permission_id INTEGER NOT NULL,
        PRIMARY KEY(group_id, permission_id),
        FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE,
        FOREIGN KEY(permission_id) REFERENCES permissions(id) ON DELETE CASCADE
    )`;
    if _, err := db.Exec(groupPermsTable); err != nil {
        return fmt.Errorf("create group_permissions: %w", err)
    }
    // Create EXPENSE_REPORTS table
    reportsTable := `CREATE TABLE IF NOT EXISTS expense_reports (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        title TEXT NOT NULL,
        status TEXT NOT NULL,
        created_at DATETIME NOT NULL,
        FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
    )`;
    if _, err := db.Exec(reportsTable); err != nil {
        return fmt.Errorf("create expense_reports: %w", err)
    }
    // Create EXPENSE_ITEMS table
    itemsTable := `CREATE TABLE IF NOT EXISTS expense_items (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        report_id INTEGER NOT NULL,
        description TEXT NOT NULL,
        expense_date DATE NOT NULL,
        amount_ht REAL NOT NULL,
        amount_ttc REAL NOT NULL,
        vat_rate REAL NOT NULL,
        receipt_path TEXT,
        created_at DATETIME NOT NULL,
        FOREIGN KEY(report_id) REFERENCES expense_reports(id) ON DELETE CASCADE
    )`;
    if _, err := db.Exec(itemsTable); err != nil {
        return fmt.Errorf("create expense_items: %w", err)
    }
    // Seed permissions and default groups
    if err := seedPermissionsAndGroups(db); err != nil {
        return err
    }
    // Seed super admin if none exists
    if err := seedSuperAdmin(db); err != nil {
        return err
    }
    return nil
}

// seedPermissionsAndGroups inserts predefined permissions and groups if they do not already exist.
func seedPermissionsAndGroups(db *sql.DB) error {
    // List of default permissions following the specification
    permissions := []string{
        "reports:create",
        "reports:update:own",
        "reports:read:own",
        "reports:read:all",
        "reports:approve",
        "reports:reject",
        "users:read",
        "users:create",
        "groups:read",
        "groups:create",
        "permissions:assign",
        "tokens:create",
        "reports:export:all",
    }
    for _, action := range permissions {
        var id int
        err := db.QueryRow("SELECT id FROM permissions WHERE action = ?", action).Scan(&id)
        if errors.Is(err, sql.ErrNoRows) {
            if _, err := db.Exec("INSERT INTO permissions (action) VALUES (?)", action); err != nil {
                return fmt.Errorf("inserting permission %s: %w", action, err)
            }
        } else if err != nil {
            return fmt.Errorf("checking permission %s: %w", action, err)
        }
    }
    // Default groups and their permissions
    type groupDef struct {
        name        string
        permissions []string
    }
    defs := []groupDef{
        {
            name: "Administrateurs",
            permissions: permissions, // all permissions
        },
        {
            name: "Validateurs",
            permissions: []string{"reports:read:all", "reports:approve", "reports:reject"},
        },
        {
            name: "Utilisateurs",
            permissions: []string{"reports:create", "reports:update:own", "reports:read:own"},
        },
    }
    for _, def := range defs {
        var groupID int64
        err := db.QueryRow("SELECT id FROM groups WHERE name = ?", def.name).Scan(&groupID)
        if errors.Is(err, sql.ErrNoRows) {
            res, err := db.Exec("INSERT INTO groups (name) VALUES (?)", def.name)
            if err != nil {
                return fmt.Errorf("insert group %s: %w", def.name, err)
            }
            groupID, _ = res.LastInsertId()
        } else if err != nil {
            return fmt.Errorf("select group %s: %w", def.name, err)
        }
        // Assign permissions
        for _, perm := range def.permissions {
            var permID int64
            if err := db.QueryRow("SELECT id FROM permissions WHERE action = ?", perm).Scan(&permID); err != nil {
                return fmt.Errorf("select permission %s: %w", perm, err)
            }
            // Check if assignment exists
            var exists int
            err := db.QueryRow("SELECT 1 FROM group_permissions WHERE group_id = ? AND permission_id = ?", groupID, permID).Scan(&exists)
            if errors.Is(err, sql.ErrNoRows) {
                if _, err := db.Exec("INSERT INTO group_permissions (group_id, permission_id) VALUES (?, ?)", groupID, permID); err != nil {
                    return fmt.Errorf("assign permission %d to group %d: %w", permID, groupID, err)
                }
            } else if err != nil {
                return fmt.Errorf("check assignment: %w", err)
            }
        }
    }
    return nil
}

// seedSuperAdmin ensures that at least one user exists. If none, it creates a default super admin
// user and assigns them to the Administrateurs group. The default credentials are read from
// environment variables ADMIN_EMAIL and ADMIN_PASSWORD, with fallbacks.
func seedSuperAdmin(db *sql.DB) error {
    var count int
    if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
        return fmt.Errorf("count users: %w", err)
    }
    if count > 0 {
        return nil
    }
	// Create default super admin user from env or fall back to defaults
	email := os.Getenv("ADMIN_EMAIL")
	if email == "" {
		email = "admin@example.com"
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "admin"
	}
    // Hash password using bcrypt
    hash, err := hashPassword(password)
    if err != nil {
        return fmt.Errorf("hash default password: %w", err)
    }
    res, err := db.Exec("INSERT INTO users (email, password_hash, created_at) VALUES (?, ?, ?)", email, hash, time.Now().UTC())
    if err != nil {
        return fmt.Errorf("insert super admin user: %w", err)
    }
    userID, _ := res.LastInsertId()
    // Assign user to Administrateurs group
    var groupID int64
    if err := db.QueryRow("SELECT id FROM groups WHERE name = ?", "Administrateurs").Scan(&groupID); err != nil {
        return fmt.Errorf("select administrateurs group: %w", err)
    }
    if _, err := db.Exec("INSERT INTO user_groups (user_id, group_id) VALUES (?, ?)", userID, groupID); err != nil {
        return fmt.Errorf("assign super admin to group: %w", err)
    }
    fmt.Printf("[INFO] Super admin created with email %s and password %s\n", email, password)
    return nil
}
