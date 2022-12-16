# Start by building the application.
FROM golang:1.18 as build

COPY src/ /go/src
WORKDIR /go/src

RUN go get github.com/newrelic/go-agent/v3/newrelic@v3.20.2
RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/bin/app

# Now copy it into our base image.
FROM gcr.io/distroless/static-debian11
COPY --from=build /go/bin/app /
EXPOSE 8080
CMD ["/app"]