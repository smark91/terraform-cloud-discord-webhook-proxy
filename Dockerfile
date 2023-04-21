FROM alpine:3.17

# Install the CA certificates needed to make HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy the built binary from goreleaser dist folder
COPY terraform-cloud-discord-webhook-proxy /app

# Start the application
ENTRYPOINT ["/app"]
