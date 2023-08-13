# Use the official Go image from the DockerHub
FROM golang:1.20 as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code into the container
COPY cmd/ cmd/
COPY internal/ internal/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ingress-annotation-modifier ./cmd/ingress-annotation-modifier

### Start a new stage from scratch ###
FROM alpine:latest  

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the pre-built binary from the previous stage
COPY --from=builder /app/ingress-annotation-modifier .

# Command to run the executable
CMD ["./ingress-annotation-modifier"]
