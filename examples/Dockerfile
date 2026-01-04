# Example Dockerfile: Injecting Trivy into Octovy container
#
# This Dockerfile demonstrates how to inject Trivy binary into the Octovy container.
# The multi-stage build copies Trivy from the official Trivy image.

# Stage 1: Get Trivy binary
FROM aquasec/trivy:0.68.1 AS trivy

# Stage 2: Octovy with Trivy injected
FROM ghcr.io/secmon-lab/octovy:latest

# Copy Trivy binary from the trivy stage
COPY --from=trivy /usr/local/bin/trivy /usr/local/bin/trivy

# Set the path to Trivy
ENV OCTOVY_TRIVY_PATH=/usr/local/bin/trivy

# The rest of the configuration is the same as the base image
# You can override other environment variables as needed:
# ENV OCTOVY_ADDR=:8080
# ENV OCTOVY_GITHUB_APP_ID=your-app-id
# etc.
