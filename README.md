# Expense App

A simple expense tracking application built with Go and Svelte.

## Getting Started

To get the application running locally, you'll need to have Docker installed. You can also use the provided `Makefile` to simplify the process.

1.  **Build and run the application:**

    ```bash
    make run
    ```

2.  **Access the application:**

    The application will be available at `http://localhost:8080`.

## Makefile Commands

The `Makefile` provides several commands to manage the application lifecycle:

*   `make build`: Builds the Docker image for the application.
*   `make run`: Starts the application in a Docker container.
*   `make stop`: Stops and removes the application container.
*   `make logs`: Tails the logs of the application container.
*   `make clean`: Stops the container and removes the Docker image.
*   `make test`: Runs both unit and functional tests.
*   `make test-unit`: Runs only the unit tests.
*   `make test-functional`: Runs only the functional tests.

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
