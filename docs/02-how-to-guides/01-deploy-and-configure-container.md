# Deploy and Configure the Container

Read [Environment Variables, Schema, and Makefile](../03-reference/02-env-vars-schema-and-makefile.md) before you start this procedure.

## Prerequisites

- Install Docker and GNU Make on the host.
- Prepare a writable directory for persistent SQLite data.

## Safety Instructions

**WARNING:** Set custom values for `JWT_SECRET` and `ADMIN_PASSWORD` before container startup. Default credentials expose administrative API routes to unauthorized callers.

## Steps

1. Build the Docker image from the repository root:

   ```bash
   make build
   ```

2. Create the local data directory on the host:

   ```bash
   mkdir -p ./data
   ```

3. Start the container with custom `JWT_SECRET`, `ADMIN_EMAIL`, and `ADMIN_PASSWORD` values:

   ```bash
   docker run -d \
     -p 8080:8080 \
     -e PORT=8081 \
     -e DATADIR=/data \
     -e JWT_SECRET="replace-with-random-secret" \
     -e ADMIN_EMAIL="admin@corp.local" \
     -e ADMIN_PASSWORD="replace-with-strong-password" \
     -v "$(pwd)/data:/data" \
     --name expenseapp \
     expenseapp
   ```

4. Wait for the Go backend and Nginx proxy inside `expenseapp` to listen on port `8080`.

## Verification

1. Check container logs and confirm the Go listener message:

   ```bash
   docker logs expenseapp
   ```

2. Send a login request to `/api/auth/login`:

   ```bash
   curl -s -X POST http://localhost:8080/api/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"admin@corp.local","password":"replace-with-strong-password"}'
   ```

3. Confirm that the JSON response contains a non-empty `token` field.

## Rollback

1. If the container fails health checks, stop and remove it immediately:

   ```bash
   make stop
   ```

2. Restore the previous `./data/expense.db` backup before you start a corrected container.
