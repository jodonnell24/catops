FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY app/. .
RUN CGO_ENABLED=0 go build -o /bin/app .
FROM gcr.io/distroless/static-debian11
WORKDIR /
COPY --from=builder /bin/app .
EXPOSE 8080
ENTRYPOINT ["/app"]
