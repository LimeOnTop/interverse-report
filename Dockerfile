# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY interverse-report/go.mod interverse-report/go.sum ./
COPY interverse-contracts /interverse-contracts
RUN go mod edit -replace=github.com/LimeOnTop/interverse-contracts=/interverse-contracts
RUN go mod download

COPY interverse-report/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o report-service .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
RUN adduser -D -s /bin/sh appuser
USER appuser
WORKDIR /app
COPY --from=builder /app/report-service .
EXPOSE 50054
CMD ["./report-service"]
