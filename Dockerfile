# Stage 1: Build the Go backend binary
FROM golang:1.20-alpine3.17 AS builder

# Install git and C build tools for go-sqlite3 CGO compilation
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy all source files
COPY . .

# Download and vendor Go dependencies
RUN go mod tidy
RUN go mod vendor

# Build the Go binary using vendored dependencies
RUN CGO_ENABLED=1 GOOS=linux go build -mod=vendor -a -installsuffix cgo -o /app/expenseapp .

# Stage 2: Create the final Nginx runtime image
FROM nginx:alpine

# Copy the Go binary from the builder stage
COPY --from=builder /app/expenseapp /usr/local/bin/expenseapp

# Copy static frontend assets
COPY --from=builder /app/frontend /usr/share/nginx/html

# Create the persistent data directory
RUN mkdir /data

# Configure backend environment variables inside container
ENV PORT=8081
ENV DATADIR=/data

# Remove default Nginx virtual host configuration
RUN rm /etc/nginx/conf.d/default.conf

# Create container entrypoint script
RUN echo '#!/bin/sh' > /entrypoint.sh && \
    echo '/usr/local/bin/expenseapp &' >> /entrypoint.sh && \
    echo 'nginx -g "daemon off;"' >> /entrypoint.sh && \
    chmod +x /entrypoint.sh

# Create Nginx reverse proxy configuration forwarding /api/ to Go backend
RUN echo 'server {' > /etc/nginx/conf.d/app.conf && \
    echo '    listen 8080;' >> /etc/nginx/conf.d/app.conf && \
    echo '    server_name localhost;' >> /etc/nginx/conf.d/app.conf && \
    echo '    location / {' >> /etc/nginx/conf.d/app.conf && \
    echo '        root   /usr/share/nginx/html;' >> /etc/nginx/conf.d/app.conf && \
    echo '        index  index.html index.htm;' >> /etc/nginx/conf.d/app.conf && \
    echo '        try_files $uri $uri/ /index.html;' >> /etc/nginx/conf.d/app.conf && \
    echo '    }' >> /etc/nginx/conf.d/app.conf && \
    echo '    location /api/ {' >> /etc/nginx/conf.d/app.conf && \
    echo '        proxy_pass http://127.0.0.1:8081/api/;' >> /etc/nginx/conf.d/app.conf && \
    echo '        proxy_set_header Host $host;' >> /etc/nginx/conf.d/app.conf && \
    echo '        proxy_set_header X-Real-IP $remote_addr;' >> /etc/nginx/conf.d/app.conf && \
    echo '        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;' >> /etc/nginx/conf.d/app.conf && \
    echo '        proxy_set_header X-Forwarded-Proto $scheme;' >> /etc/nginx/conf.d/app.conf && \
    echo '    }' >> /etc/nginx/conf.d/app.conf && \
    echo '}' >> /etc/nginx/conf.d/app.conf

# Expose HTTP port and start entrypoint
EXPOSE 8080
CMD ["/entrypoint.sh"]
