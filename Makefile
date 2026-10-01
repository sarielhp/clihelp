.PHONY: all check fix lint test build format tidy vet staticcheck map version bump commit push ci checkpoint clean run audit

all: check

audit:
	@./tools/audit.sh

check:
	@./tools/check.sh

fix:
	@./tools/fix.sh

lint:
	@./tools/lint.sh

test:
	@go test -race -timeout 300s ./...

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

# $(value ARGS) is the message exactly as it was typed. Reading ARGS any other
# way — $(ARGS) in the recipe, or the exported variable itself — makes expands
# the text first, and $5, $HOME and $(x) in a commit message disappear without
# a word ("cost $5 $HOME" is committed as "cost  OME"). The := stores the raw
# text once; export carries it to the script untouched, newlines included.
COMMIT_MESSAGE := $(value ARGS)
export COMMIT_MESSAGE
# make exports command-line variables on its own, expanding them as it does; a
# message that mentions $(ARGS) would then fail as a recursive reference.
unexport ARGS

commit:
	@./tools/commit.sh "$$COMMIT_MESSAGE"

push: bump

ci: check

checkpoint:
	@./tools/checkpoint.sh

run:
	@./tools/run_example.sh

clean:
	@go clean
	@echo "Cleaned."
