.PHONY: all build package clean test

# Ensure bin directory exists
bin:
	@mkdir -p bin

# Ensure test/reports directory exists
test/reports:
	@mkdir -p test/reports

all: build package

test: test/reports
	@echo "Running tests..."
	@./scripts/test

build: bin test
	@echo "Running build script..."
	@./scripts/build

package: build
	@echo "Running package script..."
	@./scripts/package

clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -rf test/reports/
	@docker rmi todoissh 2>/dev/null || true 