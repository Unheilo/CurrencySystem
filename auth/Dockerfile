FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY auth ./auth
COPY pkg ./pkg

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/auth ./auth/cmd

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/auth /auth
COPY auth/internal/config/config.example.yaml /etc/auth/config.yaml

ENV CONFIG_PATH=/etc/auth/config.yaml
EXPOSE 8081
USER nonroot:nonroot

ENTRYPOINT ["/auth"]
