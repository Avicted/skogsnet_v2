FROM node:24-alpine AS frontend-builder
WORKDIR /app
COPY skogsnet-frontend/ .
RUN npm ci && npm run build


FROM golang:1.24.5-alpine AS builder
WORKDIR /app
RUN apk add --no-cache gcc musl-dev
COPY . .
RUN go mod download
ENV CGO_ENABLED=1
RUN go build -o skogsnet_v2 ./internal


FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/skogsnet_v2 .
COPY --from=frontend-builder /app/dist ./skogsnet-frontend/dist
COPY entrypoint.sh .
EXPOSE 8080
ENTRYPOINT ["./entrypoint.sh"]