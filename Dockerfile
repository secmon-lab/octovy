FROM golang:1.25.5 AS build-go
ENV CGO_ENABLED=0
ARG BUILD_VERSION

WORKDIR /app
RUN go env -w GOMODCACHE=/root/.cache/go-build

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build go mod download

COPY . /app
RUN --mount=type=cache,target=/root/.cache/go-build go build -o octovy -ldflags "-X github.com/m-mizutani/octovy/pkg/domain/types.AppVersion=${BUILD_VERSION}" .

FROM gcr.io/distroless/base:nonroot
USER nonroot
COPY --from=build-go /app/octovy /octovy
COPY --from=aquasec/trivy:0.68.1 /usr/local/bin/trivy /trivy
WORKDIR /
ENV OCTOVY_ADDR="0.0.0.0:8000"
ENV OCTOVY_TRIVY_PATH=/trivy
EXPOSE 8000

ENTRYPOINT ["/octovy"]

