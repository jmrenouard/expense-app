# Docker image name
IMAGE_NAME = expenseapp

# Default target
all: build

# Build Docker image
build:
	docker build -t $(IMAGE_NAME) .

# Run Docker container
run:
	docker run -d -p 8080:8080 --name $(IMAGE_NAME) -v $(PWD)/data:/data $(IMAGE_NAME)

# Stop and remove Docker container
stop:
	docker stop $(IMAGE_NAME) || true
	docker rm $(IMAGE_NAME) || true

# Tail container logs
logs:
	docker logs -f $(IMAGE_NAME)

# Stop container and remove Docker image
clean: stop
	docker rmi $(IMAGE_NAME) || true

# Run Go unit tests
test-unit:
	@echo "Running unit tests..."
	go test -v .

# Run containerized functional tests
test-functional: build
	@echo "--> Running functional tests..."
	@sh -c ' \
		set -e; \
		NETWORK_NAME=expenseapp-test-net; \
		APP_CONTAINER_NAME=expenseapp-test-app; \
		TEST_IMAGE_NAME=$(IMAGE_NAME)-test; \
		cleanup() { \
			echo "    Cleaning up..."; \
			docker stop $$APP_CONTAINER_NAME >/dev/null 2>&1 || true; \
			docker rm $$APP_CONTAINER_NAME >/dev/null 2>&1 || true; \
			docker network rm $$NETWORK_NAME >/dev/null 2>&1 || true; \
		}; \
		trap cleanup EXIT; \
		echo "    Setting up test environment..."; \
		docker network create $$NETWORK_NAME >/dev/null 2>&1 || true; \
		docker run -d --name $$APP_CONTAINER_NAME --network $$NETWORK_NAME $(IMAGE_NAME) >/dev/null; \
		echo "    Building test image..."; \
		docker build -t $$TEST_IMAGE_NAME -f tests/Dockerfile . >/dev/null; \
		echo "    Running tests..."; \
		docker run --network $$NETWORK_NAME --rm -e API_URL=http://$$APP_CONTAINER_NAME:8080 $$TEST_IMAGE_NAME; \
		echo "    Functional tests passed."; \
	'

# Run all unit and functional tests
test: test-unit test-functional

.PHONY: all build run stop logs clean test-unit test-functional test
