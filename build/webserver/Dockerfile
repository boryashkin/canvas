FROM golang:1.20 as build

WORKDIR /go/src/app
COPY . .

RUN go mod download

WORKDIR /go/src/app/cmd/webserver
RUN CGO_ENABLED=0 go build -o /go/bin/app

FROM gcr.io/distroless/static-debian11
LABEL org.opencontainers.image.source=https://github.com/boryashkin/canvas

COPY --from=build /go/bin/app /
CMD ["/app"]