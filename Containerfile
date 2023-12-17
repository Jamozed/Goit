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
ENV XDG_CONFIG_HOME=/app/config XDG_DATA_HOME=/app/data XDG_STATE_HOME=/app/state
EXPOSE 8080
VOLUME /app/config/goit /app/data/goit /app/state/goit
ENTRYPOINT ["/app/bin/goit"]
