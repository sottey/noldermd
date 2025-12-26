FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/noldermd ./cmd/noldermd
RUN mkdir -p /notes

FROM gcr.io/distroless/static:nonroot

WORKDIR /app
COPY --from=builder /out/noldermd /app/noldermd
COPY --from=builder --chown=65532:65532 /notes /notes

EXPOSE 8080
ENTRYPOINT ["/app/noldermd", "serve", "--notes-dir", "/notes", "--port", "8080"]
