FROM golang:1.23 AS builder

WORKDIR /app

COPY . .

# use static binary for compatibality and lightweight image
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o server ./cmd/server

FROM scratch

COPY --from=builder /app/server /server

CMD ["/server"]