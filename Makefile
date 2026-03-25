.PHONY: build ui-install ui-build ui-dev test lint nilaway clean

# ── Go build ──────────────────────────────────────────────────────────────────
build: ui-build
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/vynilino ./cmd/vynilino

# ── UI tasks ─────────────────────────────────────────────────────────────────
ui-install:
	cd ui && npm ci

ui-build: ui-install
	cd ui && npm run build

ui-dev:
	cd ui && npm run dev

# ── Tests ─────────────────────────────────────────────────────────────────────
test:
	go test ./...

# ── Lint ──────────────────────────────────────────────────────────────────────
lint:
	go vet ./...
	golangci-lint run ./...

nilaway:
	go install go.uber.org/nilaway/cmd/nilaway@latest
	nilaway ./...

# ── Clean ─────────────────────────────────────────────────────────────────────
clean:
	rm -rf bin/ ui/dist/ ui/node_modules/
