# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Download dependencies first (better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o resume_agent ./cmd/resume_agent

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies: ca-certificates for HTTPS, texlive for LaTeX compilation
RUN apk add --no-cache ca-certificates texlive texlive-xetex

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/resume_agent .

# Copy templates (embedded at runtime)
COPY templates/ templates/

ENTRYPOINT ["./resume_agent"]
CMD ["serve", "--port", "8080"]
