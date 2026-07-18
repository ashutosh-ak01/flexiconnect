# Use an alpine base image for a small footprint
FROM alpine:3.19

# Install CA certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create a non-root user for security
RUN adduser -D -g '' flexiconnect

# Copy the pre-built binary from goreleaser
COPY flexiconnect /usr/local/bin/flexiconnect

# Use the non-root user
USER flexiconnect

# Expose the default port (can be overridden by config)
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/usr/local/bin/flexiconnect"]
