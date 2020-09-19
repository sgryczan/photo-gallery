FROM golang:alpine AS builder

ARG VERSION

RUN apk update && apk add --no-cache \
    git \
    curl \
    jq  \
    gcc \
    libc-dev

RUN dir=$(mktemp -d) && \
    git clone https://github.com/go-swagger/go-swagger "$dir" && \
    cd "$dir" && \
    go install ./cmd/swagger

WORKDIR $GOPATH/src/pkg/app/

RUN git clone https://github.com/swagger-api/swagger-ui && \
    mv swagger-ui/dist swaggerui && \
    sed -i 's%https://petstore.swagger.io/v2%.%g' swaggerui/index.html

COPY . .

RUN go get -d -v

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X github.com/sgryczan/photoGallery/uploader/handlers.Version=${VERSION}" -o /go/bin/api
RUN swagger generate spec --scan-models -o ./swaggerui/swagger.json

FROM alpine

RUN apk update && apk add --no-cache ca-certificates
EXPOSE 8080

COPY --from=builder /go/bin/api /go/bin/api
COPY --from=builder /go/src/pkg/app/swaggerui /go/bin/swaggerui
WORKDIR /go/bin

ENTRYPOINT ["./api"]
