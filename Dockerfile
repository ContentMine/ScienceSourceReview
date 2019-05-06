FROM golang:1.12

ADD src /go/src
ADD templates /go/templates
ADD static /go/static

ADD ./docker.json /go/config.json

RUN go get ./...

RUN go install github.com/ContentMine/ScienceSourceReview

ENTRYPOINT ["/go/bin/ScienceSourceReview", "-config", "/go/config.json"]

EXPOSE 4242
