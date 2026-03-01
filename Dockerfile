FROM golang:1.26-alpine AS build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/runbook-hunter ./cmd/runbook-hunter

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/runbook-hunter /app/runbook-hunter
COPY backend/internal/store/migrations /app/internal/store/migrations
COPY runbooks /runbooks
EXPOSE 8080
ENTRYPOINT ["/app/runbook-hunter"]
