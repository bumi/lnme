# Go base image
FROM golang:1.17-alpine as builder

RUN go get github.com/GeertJohan/go.rice && go get github.com/GeertJohan/go.rice/rice

WORKDIR /app

COPY . .

RUN go install

RUN rice embed-go && go build

FROM golang:1.17-alpine

COPY --from=builder /app/lnme /lnme

CMD ["/lnme"]
