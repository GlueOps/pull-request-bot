# ---- build stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/pr-bot .

# ---- runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

COPY --from=build /out/pr-bot /pr-bot

# No port needed; it runs as a worker
ENTRYPOINT ["/pr-bot"]