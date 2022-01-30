# Go base image
FROM golang:1.17-alpine as builder

WORKDIR /app

COPY . .

RUN go install

RUN go build

FROM alpine:3.14

COPY --from=builder /app/lnme /lnme

EXPOSE 1323

CMD ["/lnme"]
