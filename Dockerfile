FROM docker.io/golang:2.13.0-alpine AS build_base

ENV CGO_ENABLED=1
ENV GO111MODULE=on
RUN apk add --no-cache git gcc g++

# Set the Current Working Directory inside the container
WORKDIR /src

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# Init
RUN go run scripts/init/main.go -copyData="config.json,database.db,index.tmpl"

# Build the Go app
RUN go build -o ./out/app ./cmd/api/main.go

# Start fresh from a smaller image
# https://github.com/docker-library/golang/blob/8e04c39d2ce4466162418245c8b1178951021321/1.19/alpine3.17/Dockerfile
FROM docker.io/alpine:3.17
# https://mozillazg.com/2020/03/use-alpine-image-common-issues.rst.html
RUN apk --no-cache add ca-certificates && \
    update-ca-certificates
# time: missing Location in call to Time.In
# https://medium.com/freethreads/panic-time-missing-location-in-call-to-date-89d171811d3
RUN apk --no-cache add tzdata
RUN apk --no-cache add curl

WORKDIR /app

COPY --from=build_base /src/out/app /app/api
COPY --from=build_base /src/data /app/data

RUN chmod +x api

# This container exposes port 3000 to the outside world
EXPOSE 3000

# Run the binary program produced by `go install`
# Or, "data/config.secret.json"
ENTRYPOINT ./api --config-path="data/config.json"
