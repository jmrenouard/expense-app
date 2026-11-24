# Manage Users, Groups, and Approvals

Read [REST API and Permissions](../03-reference/01-rest-api-and-permissions.md) before you start this procedure.

## Prerequisites

- Confirm that `expenseapp` runs and accepts HTTP requests on `http://localhost:8080`.
- Authenticate as an `Administrateurs` member and export `$ADMIN_TOKEN`.

## Steps

1. List seeded groups to retrieve numeric group IDs (`Administrateurs`, `Validateurs`, `Utilisateurs`):

   ```bash
   curl -s -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8080/api/admin/groups
   ```

2. Create a user account assigned to group ID `3` (`Utilisateurs`):

   ```bash
   curl -s -X POST http://localhost:8080/api/admin/users \
     -H "Authorization: Bearer $ADMIN_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"email":"employee@corp.local","password":"change-me-now","groups":[3]}'
   ```

3. List submitted reports that await validator review:

   ```bash
   curl -s -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8080/api/admin/reports
   ```

4. Approve or reject a submitted report by numeric report ID:

   ```bash
   curl -s -X POST -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8080/api/admin/reports/1/approve
   ```

**NOTE:** Only reports in `submitted` status transition to `approved` or `rejected`.

## Verification

1. Run `GET /api/admin/users` and confirm that the new email address appears in the JSON list.
2. Run `GET /api/admin/reports` and verify that report `1` now shows `"status":"approved"`.

## Rollback

1. If an approval uses the wrong report ID, inspect `GET /api/admin/reports` immediately.
2. Create a corrected report or restore the pre-change SQLite backup if audit compliance requires state reset.
