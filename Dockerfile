# syntax=docker/dockerfile:1

FROM golang:1.21-bookworm AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/kubespark-server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /

COPY --from=builder /out/kubespark-server /kubespark-server

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/kubespark-server"]
CMD ["--port=8080"]
