# Start by building the application.
FROM golang:1.14-buster as build

WORKDIR /go/src/app
ADD . /go/src/app

RUN go get -d -v ./...
RUN ls
RUN go build /go/src/app/cmd/gkbm -o /go/bin/app

# Now copy it into our base image.
FROM gcr.io/distroless/base
COPY --from=build /go/bin/app /
CMD ["/gkbm"]
