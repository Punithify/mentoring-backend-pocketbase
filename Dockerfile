# Use a Debian-based Golang image to build the application
FROM golang:1.21-bullseye as builder

WORKDIR /app

# Copy go.mod and go.sum, then download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire application code
COPY . .

# Build the application, producing the mentoring_backend binary
RUN go build -o mentoring_backend main.go

# Final image
FROM gcr.io/google.com/cloudsdktool/cloud-sdk:slim

# Set working directory
WORKDIR /root/

# Add the gcsfuse repository and install gcsfuse
RUN echo "deb http://packages.cloud.google.com/apt gcsfuse-bullseye main" | tee /etc/apt/sources.list.d/gcsfuse.list \
    && curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - \
    && apt-get update \
    && apt-get install -y gcsfuse

# Copy the built application from the builder stage
COPY --from=builder /app/mentoring_backend .

# Set environment variables for Google Cloud Storage bucket and port
ENV BUCKET_NAME=my-pocketbase-data
ENV PORT 8080

# Expose the Cloud Run default port
EXPOSE 8080

# Mount the GCS bucket and start the application
CMD ["sh", "-c", "gcsfuse $BUCKET_NAME /root/data && ./mentoring_backend --dataDir=/root/data"]
