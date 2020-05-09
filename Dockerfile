# Start by building the application.
FROM golang:1.14-buster as build

WORKDIR /go/src/app
ADD . /go/src/app

RUN go get -d -v ./...

WORKDIR /go/src/app/cmd/gkbm
RUN go test
RUN GOOS=linux GOARCH=amd64 go build -v -ldflags="-w -s" -o /go/bin/app

# Now copy it into our base image.
FROM gcr.io/distroless/base
COPY --from=build /go/bin/app /
CMD ["/gkbm"]
