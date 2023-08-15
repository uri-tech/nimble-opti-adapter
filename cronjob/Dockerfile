# Stage 1: Build the application
FROM golang:1.20-alpine AS builder

# Install required packages for building the application
RUN apk update && apk add --no-cache git make musl-dev go

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files to cache dependencies
COPY go.mod go.sum ./

# Download dependencies before copying the application source code
RUN go mod download

# Copy the application code into the container
COPY cronjob/ cronjob/
COPY utils/ utils/
COPY loggerpkg/ loggerpkg/

# Build the application
RUN go build -o ingress-annotation-modifier ./cronjob/cmd/ingress-annotation-modifier

############################
# STEP 2 build a small image
############################
# FROM alpine:3.17.2
FROM alpine:latest

# Install required packages for building the application
RUN mkdir -p /myapp/src

# Install required packages for running the application. --no-cache option to avoid creating a cache in the Docker image.
RUN apk add --no-cache ca-certificates

# Create a non-root user to run the application
RUN adduser -D -g '' appuser

# Remove unnecessary packages
RUN rm -rf /var/cache/*
# RUN apk del openssh-client && rm -rf /var/cache/apk/*

# run the app as non-root user
USER appuser

# Set the working directory and copy the binary from the builder stage
WORKDIR /myapp

# Copy the binary and specific files and folders from the builder stage
# COPY --from=builder --chown=appuser:appuser /app/ingress-annotation-modifier .
COPY --from=builder /app/ingress-annotation-modifier .

# Set the command to run the binary
CMD ["/myapp/ingress-annotation-modifier"]
