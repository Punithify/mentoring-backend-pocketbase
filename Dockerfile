# Use the official Golang image to build your extended PocketBase
FROM golang:1.17-alpine as builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files to install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire application code
COPY . .

# Build your customized PocketBase application
RUN go build -o pocketbase main.go

# Final image for running the app
FROM alpine:latest

# Set the working directory and copy over the built binary
WORKDIR /root/
COPY --from=builder /app/pocketbase .

# Expose the PocketBase port (8090 by default)
EXPOSE 8090

# Start the application
CMD ["./pocketbase", "serve", "--http=0.0.0.0:8090"]
