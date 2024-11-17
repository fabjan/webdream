FROM golang:1.23 as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY dream ./dream
COPY groq ./groq
COPY metrics ./metrics
COPY public ./public
COPY webdream.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o webdream .

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/public ./public
COPY --from=builder /app/webdream .

CMD ["./webdream"]
