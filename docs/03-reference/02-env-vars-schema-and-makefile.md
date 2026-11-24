# Environment Variables, Schema, and Makefile

## Environment Variables (`main.go`, `auth.go`, `models.go`, `Dockerfile`)

| Variable | Purpose | Standalone Default | Docker Image Default |
| --- | --- | --- | --- |
| `PORT` | HTTP listen port for Gin server | `8080` | `8081` (Nginx exposes `8080`) |
| `DATADIR` | Root directory for `expense.db` and `receipts/` | `./data` | `/data` |
| `JWT_SECRET` | HS256 signing secret read at startup | `secret` | `secret` |
| `ADMIN_EMAIL` | Initial super-admin email when `users` is empty | `admin@example.com` | `admin@example.com` |
| `ADMIN_PASSWORD` | Initial super-admin password when `users` is empty | `admin` | `admin` |

## SQLite Tables (`models.go`, `handlers.go`)

- `users` (`id`, `email`, `password_hash`, `created_at`)
- `groups` (`id`, `name`)
- `permissions` (`id`, `action`)
- `user_groups` (`user_id`, `group_id`)
- `group_permissions` (`group_id`, `permission_id`)
- `expense_reports` (`id`, `user_id`, `title`, `status`, `created_at`)
- `expense_items` (`id`, `report_id`, `description`, `expense_date`, `amount_ht`, `amount_ttc`, `vat_rate`, `receipt_path`, `created_at`)
- `api_tokens` (`user_id`, `token`, `created_at` — created lazily by `GenerateAPIToken`)

## `Makefile` Targets

| Target | Behavior |
| --- | --- |
| `make build` | Run `docker build -t expenseapp .` |
| `make run` | Run detached container publishing `:8080` and mounting `$(PWD)/data:/data` |
| `make stop` | Stop and remove container `expenseapp` |
| `make logs` | Follow container stdout/stderr logs |
| `make clean` | Stop container and remove `expenseapp` image |
| `make test-unit` | Execute `go test -v .` |
| `make test-functional` | Launch container on isolated bridge network and execute `tests/Dockerfile` |
| `make test` | Run `test-unit` followed by `test-functional` |
