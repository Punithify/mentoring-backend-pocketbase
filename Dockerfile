# Use Debian for compatibility with gcsfuse
FROM golang:1.21-bullseye as builder

WORKDIR /app

# Copy go.mod and go.sum, then download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the application code
COPY . .

# Build the application
RUN go build -o mentoring_backend main.go

# Final image
FROM gcr.io/google.com/cloudsdktool/cloud-sdk:slim

# Install gcsfuse
RUN apt-get update && apt-get install -y gcsfuse

# Copy the built binary from the builder stage
WORKDIR /root/
COPY --from=builder /app/mentoring_backend .

# Set environment variables
ENV BUCKET_NAME=my-pocketbase-data
ENV PORT 8080

# Expose port 8080
EXPOSE 8080

# Mount the GCS bucket and start the application
CMD ["sh", "-c", "gcsfuse $BUCKET_NAME /root/data && ./mentoring_backend --dataDir=/root/data"]
