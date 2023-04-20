# Use the official Go image as the base image
FROM golang:1.20.3-alpine AS build

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download the dependencies
COPY go.mod ./
RUN go mod download

# Copy the rest of the application files
COPY . .

# Run unit tests
RUN go test -v ./...

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

# Use a small Alpine Linux image as the base image for the final image
FROM alpine:3.17

# Install the CA certificates needed to make HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy the built binary from the previous stage
COPY --from=build /app/app /app/app

# Set the working directory inside the container
WORKDIR /app

# Expose the port that the application listens on
EXPOSE 8080

# Start the application
CMD ["./app"]
