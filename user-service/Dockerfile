# Start with a lightweight base image
FROM golang:1.23.1-alpine

# Set environment variables
ENV GO111MODULE=on

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
RUN go build -o user-service .

# Expose the port on which the app will run
EXPOSE 50051

# Start the service
CMD ["./user-service"]