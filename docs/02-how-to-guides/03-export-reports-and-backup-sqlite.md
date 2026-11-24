# Export Reports and Back Up SQLite

Read [Storage, Proxy, and Security Posture](../04-explanation/02-storage-proxy-and-security-posture.md) before you start this procedure.

## Prerequisites

- Authenticate as an `Administrateurs` user with `reports:export:all` permission and export `$ADMIN_TOKEN`.
- Confirm shell write access to a local backup directory on the host.

## Safety Instructions

**CAUTION:** Stop active write traffic or use `sqlite3 .backup` before you copy `expense.db`. Raw file copy during concurrent writes can corrupt the SQLite file.

## Steps

1. Export all expense reports in CSV, JSON, and YAML formats:

   ```bash
   curl -s -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8080/api/admin/export/csv -o expenses.csv
   curl -s -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8080/api/admin/export/json -o expenses.json
   curl -s -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8080/api/admin/export/yaml -o expenses.yaml
   ```

2. Create an online SQLite database snapshot safely:

   ```bash
   sqlite3 ./data/expense.db ".backup './data/expense-backup.db'"
   ```

3. Archive the SQLite backup and the `./data/` receipts directory together:

   ```bash
   tar -czf expense-backup.tar.gz ./data/expense-backup.db ./data/*/receipts
   ```

## Verification

1. Verify that `expenses.csv`, `expenses.json`, and `expenses.yaml` contain non-empty report records.
2. Run an integrity check against the SQLite backup file:

   ```bash
   sqlite3 ./data/expense-backup.db "PRAGMA integrity_check;"
   ```

3. Confirm that `PRAGMA integrity_check;` prints `ok`.

## Rollback

1. If `expense.db` becomes corrupt, stop the `expenseapp` container with `make stop`.
2. Replace `./data/expense.db` with `./data/expense-backup.db` and start the container again.
