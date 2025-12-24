FROM golang:1.23-alpine AS base
WORKDIR /app
ENV CGO_ENABLED=1 GO111MODULE=on
RUN apk add --no-cache git build-base

# --- Frontend build ---
FROM node:20-alpine AS web-build
WORKDIR /app/web
COPY web/package*.json ./
RUN npm install
COPY web/ .
RUN npm run build

# --- Development with hot reload ---
FROM base AS dev
RUN go install github.com/air-verse/air@v1.52.3
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# ensure latest frontend build is available for embed if needed
COPY --from=web-build /app/web/dist ./web/dist
EXPOSE 5555
CMD ["air", "-c", ".air.toml"]

# --- Production build ---
FROM base AS builder
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-build /app/web/dist ./web/dist
RUN go build -o /app/bin/mchat ./cmd/server

FROM gcr.io/distroless/base-debian12 AS prod
WORKDIR /app
ENV APP_PORT=5555
COPY --from=builder /app/bin/mchat /app/mchat
EXPOSE 5555
CMD ["/app/mchat"]
