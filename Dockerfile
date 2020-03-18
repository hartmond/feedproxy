FROM golang:latest as builder

RUN apt update && apt install -y git ca-certificates
RUN mkdir -p /go/src/git.othrys.net/felix/feedproxy
WORKDIR /go/src/git.othrys.net/felix/feedproxy
ADD . /go/src/git.othrys.net/felix/feedproxy
RUN go get ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o feedproxy .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/git.othrys.net/felix/feedproxy/feedproxy /app/
WORKDIR /app
EXPOSE 8889

CMD ["./feedproxy"]
