FROM golang:1.20-alpine

WORKDIR /app

# Copy Go module manifests and fetch dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy functional test suite
COPY tests/ ./tests/

# Run functional tests against the running app container
CMD ["go", "test", "-v", "./tests"]
