# Build context is the repo root (needed for go.mod/go.sum and the worker/ tree).
# On Railway: set Dockerfile Path = worker/Dockerfile, root directory = repo root.
FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY worker ./worker

RUN CGO_ENABLED=0 GOOS=linux go build -o /worker ./worker/cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /worker /worker

ENTRYPOINT ["/worker"]
