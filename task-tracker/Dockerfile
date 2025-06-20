# syntax=docker/dockerfile:1

# Stage 1: Build binary
FROM golang:1.23 AS builder

WORKDIR /app

# Copy go.mod và go.sum trước để cache dependency
COPY go.mod go.sum ./
RUN go mod download

# Copy toàn bộ source code
COPY . .

# Build file main
RUN go build -o main ./main.go


# Stage 2: Run minimal container
FROM golang:1.23

WORKDIR /app

# Copy file binary từ builder stage
COPY --from=builder /app/main .
RUN ls -l /app
# Copy logs directory (nếu có logging vào file)
RUN mkdir -p logs

# Expose port 8080
EXPOSE 8080

# Run binary
CMD ["/app/main"]
