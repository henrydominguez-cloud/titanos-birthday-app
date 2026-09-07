# ---- build stage -----------------------------------------------------------
FROM golang:1.23-alpine AS build
WORKDIR /src

# Resolve dependencies first for better layer caching. `go mod tidy` also
# generates go.sum on the fly so the repo stays lean.
COPY go.mod ./
RUN go mod tidy

COPY . .
# Static, stripped binary. CGO off so it runs on distroless/scratch.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags="-s -w" \
    -o /out/server ./cmd/server

# ---- runtime stage ---------------------------------------------------------
# Distroless static: no shell, no package manager, runs as non-root uid 65532.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/server /server
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/server"]
