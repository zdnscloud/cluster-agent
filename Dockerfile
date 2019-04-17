FROM golang:alpine AS build

RUN mkdir -p /go/src/github.com/zdnscloud/cluster-agent
COPY . /go/src/github.com/zdnscloud/cluster-agent

WORKDIR /go/src/github.com/zdnscloud/cluster-agent
RUN CGO_ENABLED=0 GOOS=linux go build -o storage/storagemanager storage/main.go


FROM alpine

LABEL maintainers="Zdns Authors"
LABEL description="Storage Manager"

COPY --from=build /go/src/github.com/zdnscloud/cluster-agent/storage/storagemanager /storagemanager

ENTRYPOINT ["/storagemanager"]
