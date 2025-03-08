.PHONY: all build package clean test test-integration test-all

all: test-all build package

# Ensure bin directory exists
bin:
	@mkdir -p bin

build: bin test
	@echo "Running build script..."
	@./scripts/build

# Build the multi-architecture Docker image and push to Docker Hub
package: test
	@echo "Building multi-architecture Docker image..."
	@./scripts/package

# Ensure test/reports directory exists
test/reports:
	@mkdir -p test/reports

test: test/reports
	@echo "Running tests..."
	@./scripts/test

test-integration: test/reports
	@echo "Running integration tests..."
	@go test -v -count=1 ./test/integration/... -coverprofile=test/reports/integration-coverage.out 2>&1 | tee test/reports/integration-test.log
	@go tool cover -html=test/reports/integration-coverage.out -o test/reports/integration-coverage.html

test-all: test test-integration

clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -rf test/reports/
	@docker rmi starbops/todoissh:latest 2>/dev/null || true