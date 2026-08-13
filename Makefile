.PHONY: all check lint test build format tidy vet staticcheck map version bump commit push ci checkpoint clean run

all: check

check:
	@./scripts/check.sh

lint:
	@./scripts/lint.sh

test:
	@go test -timeout 30s ./...

build:
	@go build -o /dev/null ./example

format:
	@./scripts/format.sh

tidy:
	@go mod tidy

vet:
	@go vet ./...

staticcheck:
	@staticcheck ./...

map:
	@./scripts/map.sh

version:
	@./scripts/version.sh

bump:
	@./scripts/bump-version.sh

commit:
	@./scripts/commit.sh $(ARGS)

push: bump

ci: check

checkpoint:
	@./scripts/checkpoint.sh

run:
	@./scripts/run_example.sh

clean:
	@go clean
	@echo "Cleaned."
