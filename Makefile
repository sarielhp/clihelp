.PHONY: all check lint test build format tidy vet staticcheck map version bump commit push ci checkpoint clean run

all: check

check:
	@./tools/check.sh

lint:
	@./tools/lint.sh

test:
	@go test -timeout 30s ./...

build:
	@go build -o /dev/null ./example

format:
	@./tools/format.sh

tidy:
	@go mod tidy

vet:
	@go vet ./...

staticcheck:
	@staticcheck ./...

map:
	@./tools/map.sh

version:
	@./tools/version.sh

bump:
	@./tools/bump-version.sh

commit:
	@./tools/commit.sh $(ARGS)

push: bump

ci: check

checkpoint:
	@./tools/checkpoint.sh

run:
	@./tools/run_example.sh

clean:
	@go clean
	@echo "Cleaned."
