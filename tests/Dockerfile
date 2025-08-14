FROM golang:1.20-alpine

WORKDIR /app

# Copy test files
COPY . .

# Copy go.mod from parent directory
COPY ../go.mod .

# Install dependencies
RUN go mod tidy

# Run the tests
CMD ["go", "test", "-v", "."]
