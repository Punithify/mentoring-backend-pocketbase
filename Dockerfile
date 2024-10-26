# Use the official Golang image with a compatible version
FROM golang:1.21-alpine as builder

WORKDIR /app

# Copy go.mod and go.sum, then download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire application code
COPY . .

# Build the application
RUN go build -o pocketbase main.go

# Final image
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/pocketbase .

EXPOSE 8090

CMD ["./pocketbase", "serve", "--http=0.0.0.0:8090"]
