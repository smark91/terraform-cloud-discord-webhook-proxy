FROM scratch
COPY terraform-cloud-discord-webhook-proxy /app
ENTRYPOINT ["/app"]
