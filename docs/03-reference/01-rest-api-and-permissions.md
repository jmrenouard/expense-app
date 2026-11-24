# REST API and Permissions

## Implemented HTTP Routes (`main.go`)

| Method | Path | Handler | Required Permission |
| --- | --- | --- | --- |
| `POST` | `/api/auth/login` | `Login` | Public (`bcrypt` check + JWT issue) |
| `POST` | `/api/reports` | `CreateReport` | `reports:create` |
| `GET` | `/api/reports` | `ListOwnReports` | `reports:read:own` |
| `POST` | `/api/reports/:id/submit` | `SubmitReport` | `reports:update:own` (owner + `draft` only) |
| `DELETE` | `/api/reports/:id` | `DeleteReport` | `reports:update:own` (owner + `draft` only) |
| `POST` | `/api/reports/:id/items` | `AddItem` | `reports:create` (owner + `draft` only) |
| `PUT` | `/api/items/:id` | `UpdateItem` | `reports:update:own` (owner + `draft` only) |
| `POST` | `/api/items/:id/receipt` | `UploadReceipt` | `reports:update:own` (owner + `draft` only) |
| `GET` | `/api/items/:id/receipt` | `GetReceipt` | `reports:read:own` (+ `reports:read:all` for non-owners) |
| `GET` | `/api/admin/reports` | `AdminListReports` | `reports:read:all` (excludes `draft`) |
| `POST` | `/api/admin/reports/:id/approve` | `ApproveReport` | `reports:approve` (`submitted` only) |
| `POST` | `/api/admin/reports/:id/reject` | `RejectReport` | `reports:reject` (`submitted` only) |
| `GET` | `/api/admin/users` | `ListUsers` | `users:read` |
| `POST` | `/api/admin/users` | `CreateUser` | `users:create` |
| `GET` | `/api/admin/groups` | `ListGroups` | `groups:read` |
| `POST` | `/api/admin/groups` | `CreateGroup` | `groups:create` |
| `POST` | `/api/admin/groups/:id/permissions` | `AssignPermissions` | `permissions:assign` |
| `POST` | `/api/admin/users/:id/token` | `GenerateAPIToken` | `tokens:create` |
| `GET` | `/api/admin/export/csv` | `ExportCSV` | `reports:export:all` |
| `GET` | `/api/admin/export/json` | `ExportJSON` | `reports:export:all` |
| `GET` | `/api/admin/export/yaml` | `ExportYAML` | `reports:export:all` |

## Seeded Groups and Permissions (`models.go`)

| Seeded Group Name | Assigned Permissions |
| --- | --- |
| `Administrateurs` | All 13 permissions (`reports:*`, `users:*`, `groups:*`, `permissions:assign`, `tokens:create`) |
| `Validateurs` | `reports:read:all`, `reports:approve`, `reports:reject` |
| `Utilisateurs` | `reports:create`, `reports:update:own`, `reports:read:own` |
