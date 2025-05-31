# Stage 1: Build the Go application
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code
COPY . .

# Build the application
# CGO_ENABLED=0 is important for a truly static binary if not using alpine base for final image,
# but good practice.
# -o /app/agbaravoip_server makes the binary output to /app/agbaravoip_server in this stage
RUN CGO_ENABLED=0 go build -o /app/agbaravoip_server ./cmd/agbaravoip_server

# Stage 2: Create the final lightweight image
FROM alpine:latest

WORKDIR /root/

# Copy the built binary from the builder stage
COPY --from=builder /app/agbaravoip_server .

# Copy the config.yml file
# This assumes config.yml is in the same directory as the Dockerfile at build time
COPY config.yml .

# Expose the server port (as defined in config.yml, default 8080)
# This is for documentation; actual port mapping is done in docker run or docker-compose
EXPOSE 8080

# Command to run the application
# The application will look for config.yml in its working directory
CMD ["./agbaravoip_server"]
