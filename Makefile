.PHONY: all build package clean

all: build package

build:
	@echo "Running build script..."
	@./scripts/build

package:
	@echo "Running package script..."
	@./scripts/package

clean:
	@echo "Cleaning up..."
	@rm -f todoissh
	@docker rmi todoissh 2>/dev/null || true 