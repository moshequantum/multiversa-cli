# multiversa-cli — developer tasks
#
# This Makefile is intentionally tiny. Real release engineering lives
# in .goreleaser.yml; this file just covers the day-to-day loop.

BINARY := multiversa
BUILD_DIR := dist
SKILL_SCRIPTS := $(HOME)/.claude/skills/lab-setup/scripts
EMBED_SCRIPTS := internal/embedded/scripts
EMBED_LIST := setup_multiversa.sh encrypted_usb_linux.sh encrypted_usb_macos.sh

.PHONY: build
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/multiversa

.PHONY: install
install:
	go install ./cmd/multiversa

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: detect stack workspace usb
detect:
	go run ./cmd/multiversa detect

stack:
	go run ./cmd/multiversa stack

workspace:
	go run ./cmd/multiversa workspace

usb:
	go run ./cmd/multiversa usb

# sync-scripts pulls the latest bash scripts from the lab-setup skill
# into internal/embedded/scripts. Run this after editing the skill
# scripts and before committing — `go:embed` reads from the file
# system at build time, so the two trees can drift silently otherwise.
.PHONY: sync-scripts
sync-scripts:
	@mkdir -p $(EMBED_SCRIPTS)
	@for s in $(EMBED_LIST); do \
		if [ -f "$(SKILL_SCRIPTS)/$$s" ]; then \
			cp "$(SKILL_SCRIPTS)/$$s" "$(EMBED_SCRIPTS)/$$s"; \
			echo "  ✓ $$s"; \
		else \
			echo "  ⚠ $$s — not in $(SKILL_SCRIPTS), skipping"; \
		fi; \
	done

# verify-scripts compares embedded vs skill scripts and exits non-zero
# if any pair has diverged. CI should run this before release.
.PHONY: verify-scripts
verify-scripts:
	@status=0; \
	for s in $(EMBED_LIST); do \
		if ! diff -q "$(EMBED_SCRIPTS)/$$s" "$(SKILL_SCRIPTS)/$$s" >/dev/null 2>&1; then \
			echo "  ✗ $$s — embedded and skill copy differ"; \
			status=1; \
		fi; \
	done; \
	if [ $$status -eq 0 ]; then echo "  ✓ all embedded scripts match the skill"; fi; \
	exit $$status

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
