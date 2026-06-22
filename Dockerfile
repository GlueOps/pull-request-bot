# ---- build stage ----
FROM golang:1.26-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648 AS builder

WORKDIR /src

COPY . .
RUN go mod tidy

RUN go build -trimpath -ldflags="-s -w" -o /out/pr-bot .

# ---- runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot@sha256:d093aa3e30dbadd3efe1310db061a14da60299baff8450a17fe0ccc514a16639

WORKDIR /

COPY --from=builder /out/pr-bot /pr-bot

# No port needed; it runs as a worker
ENTRYPOINT ["/pr-bot"]
