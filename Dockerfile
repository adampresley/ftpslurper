# Start from golang base image
FROM golang:1.22-bullseye as builder

ARG GITHUB_TOKEN

# Set the current working directory inside the container
WORKDIR /build
# RUN git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

# Copy go.mod, go.sum files and download deps
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Copy sources to the working directory and build
COPY . .
RUN echo "Building app" && make build

# Start a new stage from debian
FROM debian:bullseye
LABEL org.opencontainers.image.source=https://github.com/adampresley/ftpslurper

WORKDIR /dist

RUN apt-get update -y && apt-get install -y ca-certificates && update-ca-certificates

# Copy the build artifacts from the previous stage
COPY --from=builder /build/ftpslurper .

# Run the executable
ENTRYPOINT ["./ftpslurper"]



