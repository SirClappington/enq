# ---- builder ----
FROM golang:1.23-alpine3.22 AS builder
WORKDIR /app

# Build deps (builder only)
RUN apk add --no-cache ca-certificates git

# Leverage caching for deps
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Static build (no CGO). -buildvcs=false avoids embedding VCS metadata.
ENV CGO_ENABLED=0
ARG VERSION=""
ARG COMMIT=""
RUN go build -trimpath -buildvcs=false \
  -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -extldflags '-static'" \
  -o /bin/enq ./cmd/api

# ---- runtime ----
# Distroless base includes CA certs; good for HTTPS.
FROM gcr.io/distroless/base-debian12

# Run as non-root
USER 65532:65532

# Copy binary only
COPY --from=builder /bin/enq /enq

EXPOSE 8080
ENTRYPOINT ["/enq"]
