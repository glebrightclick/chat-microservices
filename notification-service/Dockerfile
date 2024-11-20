# Start from Go image for building the API Gateway
FROM golang:1.23.1-alpine AS builder

# Set environment variables
ENV GO111MODULE=on

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum for dependency installation
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the entire project
COPY . .

# Build the application
RUN go build -o notification-service .

EXPOSE 8182

# Run the API Gateway
CMD ["./notification-service"]
