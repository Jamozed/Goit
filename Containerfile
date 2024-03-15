FROM golang:alpine as build
RUN apk update
RUN apk upgrade
RUN apk add --no-cache build-base
COPY . /app
WORKDIR /app
ARG version
RUN VERSION=$version make build

FROM alpine:latest
RUN apk update
RUN apk upgrade
RUN apk add --no-cache git
COPY --from=build /app/bin /app/bin
RUN mkdir -p /run/user/0
WORKDIR /app
EXPOSE 8080
VOLUME /etc/goit /var/lib/goit /var/log/goit
ENTRYPOINT ["/app/bin/goit"]
