package main

import (
    "database/sql"
    "fmt"
)

// Constants representing predefined permission actions.
const (
    PermReportsCreate    = "reports:create"
    PermReportsUpdateOwn = "reports:update:own"
    PermReportsReadOwn   = "reports:read:own"
    PermReportsReadAll   = "reports:read:all"
    PermReportsApprove   = "reports:approve"
    PermReportsReject    = "reports:reject"
    PermUsersRead        = "users:read"
    PermUsersCreate      = "users:create"
    PermGroupsRead       = "groups:read"
    PermGroupsCreate     = "groups:create"
    PermPermissionsAssign = "permissions:assign"
    PermTokensCreate      = "tokens:create"
    PermReportsExportAll  = "reports:export:all"
)

// GetUserPermissions returns a set of permission actions for a user by
// aggregating permissions from all groups the user belongs to.
func GetUserPermissions(db *sql.DB, userID int64) (map[string]bool, error) {
    query := `SELECT p.action FROM permissions p
        JOIN group_permissions gp ON gp.permission_id = p.id
        JOIN user_groups ug ON ug.group_id = gp.group_id
        WHERE ug.user_id = ?`
    rows, err := db.Query(query, userID)
    if err != nil {
        return nil, fmt.Errorf("query user permissions: %w", err)
    }
    defer rows.Close()
    perms := make(map[string]bool)
    for rows.Next() {
        var action string
        if err := rows.Scan(&action); err != nil {
            return nil, fmt.Errorf("scan permission: %w", err)
        }
        perms[action] = true
    }
    return perms, nil
}

// UserHasPermission determines whether a user possesses a given permission action.
// It queries the database for the user's permissions and checks for existence.
func UserHasPermission(db *sql.DB, userID int64, action string) (bool, error) {
    perms, err := GetUserPermissions(db, userID)
    if err != nil {
        return false, err
    }
    return perms[action], nil
}
