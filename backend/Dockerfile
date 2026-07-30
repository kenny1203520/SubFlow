FROM golang:1.26.5-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/subflow ./cmd/subflow

FROM alpine:3.22 AS runner
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /pb
COPY --from=builder /out/subflow /pb/subflow
RUN addgroup -S subflow && adduser -S subflow -G subflow && mkdir -p /pb/pb_data && chown -R subflow:subflow /pb
USER subflow
EXPOSE 8090
HEALTHCHECK --interval=30s --timeout=3s --start-period=8s --retries=5 CMD wget -q --spider http://127.0.0.1:8090/api/health || exit 1
ENTRYPOINT ["/pb/subflow"]
CMD ["serve", "--http=0.0.0.0:8090", "--dir=/pb/pb_data"]
