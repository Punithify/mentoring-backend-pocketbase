# Use the official Golang image with a compatible version
FROM golang:1.21-alpine as builder

WORKDIR /app

# Copy go.mod and go.sum, then download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire application code
COPY . .

# Build the application, producing the mentoring_backend binary
RUN go build -o mentoring_backend main.go

# Final image
FROM alpine:latest
WORKDIR /root/

# Install gcsfuse for mounting Google Cloud Storage
RUN apk add --no-cache gcsfuse

# Copy the built application
COPY --from=builder /app/mentoring_backend .

# Set environment variables for Google Cloud Storage bucket and port
ENV BUCKET_NAME=my-pocketbase-data
ENV PORT 8080

# Expose the Cloud Run default port
EXPOSE 8080

# Mount the GCS bucket as a directory using gcsfuse
CMD ["sh", "-c", "gcsfuse $BUCKET_NAME /root/data && ./mentoring_backend --dataDir=/root/data"]
