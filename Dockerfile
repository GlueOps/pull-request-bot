# ---- build stage ----
FROM golang:1.26-alpine@sha256:d4c4845f5d60c6a974c6000ce58ae079328d03ab7f721a0734277e69905473e5 AS builder

WORKDIR /src

COPY . .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/pr-bot .

# ---- runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot@sha256:a9329520abc449e3b14d5bc3a6ffae065bdde0f02667fa10880c49b35c109fd1

WORKDIR /

COPY --from=build /out/pr-bot /pr-bot

# No port needed; it runs as a worker
ENTRYPOINT ["/pr-bot"]
