FROM golang:latest as builder

RUN apt update && apt install -y git ca-certificates
RUN mkdir /project
WORKDIR /project
ADD . /project
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o feedproxy .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /project/feedproxy /app/
WORKDIR /app
EXPOSE 8889

CMD ["./feedproxy"]
