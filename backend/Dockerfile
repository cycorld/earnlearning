FROM golang:1.24-alpine AS builder
WORKDIR /app
RUN apk add --no-cache gcc musl-dev
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
ARG BUILD_NUMBER=dev
ARG COMMIT_SHA=local
RUN CGO_ENABLED=1 GOOS=linux go build \
    -tags "sqlite_fts5" \
    -ldflags "-X main.BuildNumber=${BUILD_NUMBER} -X main.CommitSHA=${COMMIT_SHA}" \
    -o server ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates sqlite-libs
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/internal/infrastructure/persistence/migrations ./internal/infrastructure/persistence/migrations
COPY --from=builder /app/docs ./docs
# LLM Wiki (#071) — chatbot RAG 지식베이스. 빌드 컨텍스트는 프로젝트 루트.
COPY docs/llm-wiki ./docs/llm-wiki
EXPOSE 8080
CMD ["./server"]
