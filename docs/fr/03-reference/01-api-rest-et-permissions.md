# API REST et permissions

## Routes HTTP implémentées (`main.go`)

| Méthode | Chemin | Handler | Permission requise |
| --- | --- | --- | --- |
| `POST` | `/api/auth/login` | `Login` | Publique (vérification `bcrypt` + émission JWT) |
| `POST` | `/api/reports` | `CreateReport` | `reports:create` |
| `GET` | `/api/reports` | `ListOwnReports` | `reports:read:own` |
| `POST` | `/api/reports/:id/submit` | `SubmitReport` | `reports:update:own` (propriétaire + `draft`) |
| `DELETE` | `/api/reports/:id` | `DeleteReport` | `reports:update:own` (propriétaire + `draft`) |
| `POST` | `/api/reports/:id/items` | `AddItem` | `reports:create` (propriétaire + `draft`) |
| `PUT` | `/api/items/:id` | `UpdateItem` | `reports:update:own` (propriétaire + `draft`) |
| `POST` | `/api/items/:id/receipt` | `UploadReceipt` | `reports:update:own` (propriétaire + `draft`) |
| `GET` | `/api/items/:id/receipt` | `GetReceipt` | `reports:read:own` (+ `reports:read:all` tiers) |
| `GET` | `/api/admin/reports` | `AdminListReports` | `reports:read:all` (exclut `draft`) |
| `POST` | `/api/admin/reports/:id/approve` | `ApproveReport` | `reports:approve` (`submitted` uniquement) |
| `POST` | `/api/admin/reports/:id/reject` | `RejectReport` | `reports:reject` (`submitted` uniquement) |
| `GET` | `/api/admin/users` | `ListUsers` | `users:read` |
| `POST` | `/api/admin/users` | `CreateUser` | `users:create` |
| `GET` | `/api/admin/groups` | `ListGroups` | `groups:read` |
| `POST` | `/api/admin/groups` | `CreateGroup` | `groups:create` |
| `POST` | `/api/admin/groups/:id/permissions` | `AssignPermissions` | `permissions:assign` |
| `POST` | `/api/admin/users/:id/token` | `GenerateAPIToken` | `tokens:create` |
| `GET` | `/api/admin/export/csv` | `ExportCSV` | `reports:export:all` |
| `GET` | `/api/admin/export/json` | `ExportJSON` | `reports:export:all` |
| `GET` | `/api/admin/export/yaml` | `ExportYAML` | `reports:export:all` |

## Groupes initiaux et permissions (`models.go`)

| Nom du groupe | Permissions assignées |
| --- | --- |
| `Administrateurs` | Les 13 permissions (`reports:*`, `users:*`, `groups:*`, `permissions:assign`, `tokens:create`) |
| `Validateurs` | `reports:read:all`, `reports:approve`, `reports:reject` |
| `Utilisateurs` | `reports:create`, `reports:update:own`, `reports:read:own` |
