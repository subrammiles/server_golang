# Build from the REPOSITORY ROOT:
#   docker build -f 20-production/Dockerfile --build-arg VERSION=1.0.0 -t go-production .
#   docker run -p 8080:8080 -p 9091:9091 go-production
#
# Stage 1: compile. CGO_ENABLED=0 gives a STATIC binary that needs no C libraries at runtime.
FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download                     # cached layer: re-runs only when dependencies change
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X github.com/subrammiles/server_golang/20-production.version=${VERSION}" \
    -o /out/server ./20-production/cmd/server

# Stage 2: run. Distroless "static" has no shell, no package manager: tiny and a small attack surface.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
USER nonroot:nonroot
EXPOSE 8080 9091
ENTRYPOINT ["/server"]
