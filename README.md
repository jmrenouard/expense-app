# Expense App

A simple expense tracking application built with Go and Svelte.

## Getting Started

To get the application running locally, you'll need to have Docker and Docker Compose installed.

1.  **Build and run the application:**

    ```bash
    docker-compose up --build
    ```

2.  **Access the application:**

    The application will be available at `http://localhost:8080`.

## Configuration

The application can be configured using environment variables.

| Variable           | Description                                                                                                | Default               |
| ------------------ | ---------------------------------------------------------------------------------------------------------- | --------------------- |
| `PORT`             | The port on which the Go backend server will listen.                                                       | `8081`                |
| `DATADIR`          | The directory where the SQLite database and other data will be stored.                                     | `./data`              |
| `JWT_SECRET`       | The secret key used to sign JSON Web Tokens.                                                               | `secret`              |
| `ADMIN_EMAIL`      | The email for the initial super admin user, created on first run.                                          | `admin@example.com`   |
| `ADMIN_PASSWORD`   | The password for the initial super admin user. **It is strongly recommended to change this.**                | `admin`               |

### Example with `docker run`

You can also run the application without Docker Compose by passing the environment variables directly to the `docker run` command.

```bash
docker run -d \
  -p 8080:8080 \
  -e ADMIN_EMAIL=myadmin@example.com \
  -e ADMIN_PASSWORD=supersecretpassword \
  -e JWT_SECRET=anothersecret \
  -v $(pwd)/data:/data \
  --name expense-app \
  expense-app:latest
```

**Note:** You would first need to build the image, for example with `docker build -t expense-app:latest .`.
