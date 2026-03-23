FROM golang:1.24-alpine AS builder
WORKDIR /app
RUN apk add --no-cache gcc musl-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG BUILD_NUMBER=dev
ARG COMMIT_SHA=local
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X main.BuildNumber=${BUILD_NUMBER} -X main.CommitSHA=${COMMIT_SHA}" \
    -o server ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates sqlite-libs
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/internal/infrastructure/persistence/migrations ./internal/infrastructure/persistence/migrations
COPY --from=builder /app/docs ./docs
EXPOSE 8080
CMD ["./server"]
