# Variables, schéma et Makefile

## Variables d'environnement (`main.go`, `auth.go`, `models.go`, `Dockerfile`)

| Variable | Rôle | Défaut local | Défaut Docker |
| --- | --- | --- | --- |
| `PORT` | Port TCP d'écoute du serveur Gin | `8080` | `8081` (Nginx expose `8080`) |
| `DATADIR` | Répertoire contenant `expense.db` et `receipts/` | `./data` | `/data` |
| `JWT_SECRET` | Secret HMAC HS256 lu au démarrage | `secret` | `secret` |
| `ADMIN_EMAIL` | Email super-admin créé si `users` est vide | `admin@example.com` | `admin@example.com` |
| `ADMIN_PASSWORD` | Mot de passe super-admin créé si `users` est vide | `admin` | `admin` |

## Tables SQLite (`models.go`, `handlers.go`)

- `users` (`id`, `email`, `password_hash`, `created_at`)
- `groups` (`id`, `name`)
- `permissions` (`id`, `action`)
- `user_groups` (`user_id`, `group_id`)
- `group_permissions` (`group_id`, `permission_id`)
- `expense_reports` (`id`, `user_id`, `title`, `status`, `created_at`)
- `expense_items` (`id`, `report_id`, `description`, `expense_date`, `amount_ht`, `amount_ttc`, `vat_rate`, `receipt_path`, `created_at`)
- `api_tokens` (`user_id`, `token`, `created_at` — créée à la demande par `GenerateAPIToken`)

## Cibles `Makefile`

| Cible | Comportement |
| --- | --- |
| `make build` | Exécute `docker build -t expenseapp .` |
| `make run` | Lance le conteneur en tâche de fond sur `:8080` avec `$(PWD)/data:/data` |
| `make stop` | Arrête et supprime le conteneur `expenseapp` |
| `make logs` | Affiche les journaux du conteneur |
| `make clean` | Arrête le conteneur et supprime l'image `expenseapp` |
| `make test-unit` | Exécute `go test -v .` |
| `make test-functional` | Crée un réseau bridge temporaire et exécute `tests/Dockerfile` |
| `make test` | Enchaîne `test-unit` puis `test-functional` |
