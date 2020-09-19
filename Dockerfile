FROM golang:alpine AS builder

ARG VERSION

RUN apk update && apk add --no-cache \
    git \
    curl \
    jq  \
    gcc \
    libc-dev \
    g++

RUN mkdir $HOME/src && \
    cd $HOME/src && \
    git clone https://github.com/gohugoio/hugo.git && \
    cd hugo && \
    go install --tags extended

WORKDIR $GOPATH/src/pkg/app/

COPY gallery gallery/
COPY updater .

RUN go get -d -v

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /go/bin/api

FROM alpine

RUN apk update && apk add --no-cache ca-certificates gcc
EXPOSE 8080

COPY --from=builder /go/src/pkg/app/gallery /gallery
COPY --from=builder /go/bin/api /gallery/api
COPY --from=builder /go/bin/hugo /bin/hugo
RUN hugo version

WORKDIR /gallery

ENTRYPOINT ["./api"]
