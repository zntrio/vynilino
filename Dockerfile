# ── Stage 1: Build UI ─────────────────────────────────────────────────────────
FROM node:22-alpine AS ui-builder

WORKDIR /src/ui
COPY ui/package.json ui/package-lock.json* ./
RUN npm ci
COPY ui/ ./
RUN npm run build

# ── Stage 2: Build Go binary ──────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /src/ui/dist ./ui/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /vynilino ./cmd/vynilino

# ── Stage 3: Runtime ──────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
COPY --from=builder /vynilino /vynilino
EXPOSE 8080
ENTRYPOINT ["/vynilino", "serve"]
