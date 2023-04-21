FROM alpine:latest as certs

# Install the CA certificates needed to make HTTPS requests
RUN apk --no-cache add ca-certificates

FROM scratch

# Copy ca-certificates from alpine image
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy the built binary from goreleaser dist folder
COPY terraform-cloud-discord-webhook-proxy /app

# Start the application
ENTRYPOINT ["/app"]
