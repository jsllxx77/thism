# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN go build -o /thism-server ./cmd/server

# Stage 3: Minimal runtime image
FROM alpine:3.19
RUN adduser -D thism
COPY --from=builder /thism-server /usr/local/bin/thism-server
USER thism
ENTRYPOINT ["thism-server"]
