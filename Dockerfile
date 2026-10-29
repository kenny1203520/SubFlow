FROM dhi.io/node:26-alpine-dev AS frontend-builder
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --ignore-scripts
COPY frontend/ ./
RUN npm run build
RUN node -p "require('./package.json').version" > /src/frontend/app-version.txt

FROM dhi.io/golang:1.27.3-alpine3.24-dev AS backend-builder
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend-builder /src/frontend/app-version.txt ./internal/upgrade/version.txt
RUN rm -rf ./internal/web/dist && mkdir -p ./internal/web/dist
COPY --from=frontend-builder /src/frontend/dist/ ./internal/web/dist/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/subflow ./cmd/subflow
RUN mkdir -p /out/pb_data

FROM dhi.io/alpine-base:3.24-dev AS runtime-deps
RUN apk add --no-cache tzdata

FROM dhi.io/alpine-base:3.24 AS runtime
COPY --from=runtime-deps /usr/share/zoneinfo /usr/share/zoneinfo
WORKDIR /app
COPY --from=backend-builder --chown=65532:65532 /out/subflow /app/subflow
COPY --from=backend-builder --chown=65532:65532 /out/pb_data /app/pb_data
COPY --chmod=755 entrypoint.sh /app/entrypoint.sh
USER 65532:65532
ENV SUBFLOW_HTTP_PORT=8080
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10m --retries=5 CMD ["/app/subflow", "healthcheck"]
ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["serve", "--dir=/app/pb_data"]
