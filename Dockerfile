FROM dhi.io/node:26-alpine-dev AS frontend-builder
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM dhi.io/golang:1-alpine-dev AS backend-builder
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN rm -rf ./internal/web/dist && mkdir -p ./internal/web/dist
COPY --from=frontend-builder /src/frontend/dist/ ./internal/web/dist/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/subflow ./cmd/subflow

FROM dhi.io/alpine-base:3.24 AS runtime
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend-builder /out/subflow /app/subflow
RUN addgroup -S subflow && adduser -S subflow -G subflow && mkdir -p /app/pb_data && chown -R subflow:subflow /app
USER subflow
ENV SUBFLOW_HTTP_PORT=8080
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=8s --retries=5 CMD wget -q --spider "http://127.0.0.1:${SUBFLOW_HTTP_PORT:-8080}/api/health" || exit 1
ENTRYPOINT ["/app/subflow"]
CMD ["serve", "--dir=/app/pb_data"]
