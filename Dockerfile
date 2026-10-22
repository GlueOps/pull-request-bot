# ---- build stage ----
FROM golang:1.27-alpine@sha256:d530cf60723abf1ffae0cd16bb546b379ec0550d2f3815e5b16ed69dbc00fba4 AS builder

WORKDIR /src

COPY . .
RUN go mod tidy

RUN go build -trimpath -ldflags="-s -w" -o /out/pr-bot .

# ---- runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

WORKDIR /

COPY --from=builder /out/pr-bot /pr-bot

# No port needed; it runs as a worker
ENTRYPOINT ["/pr-bot"]
