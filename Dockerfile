FROM golang:alpine AS build

RUN mkdir -p /go/src/github.com/zdnscloud/cluster-agent
COPY . /go/src/github.com/zdnscloud/cluster-agent

WORKDIR /go/src/github.com/zdnscloud/cluster-agent
RUN CGO_ENABLED=0 GOOS=linux go build -o cmd/cluster-agent cmd/cluster-agent.go


FROM alpine

LABEL maintainers="Zdns Authors"
LABEL description="K8S Cluster Agent"

COPY --from=build /go/src/github.com/zdnscloud/cluster-agent/cmd/cluster-agent /cluster-agent

ENTRYPOINT ["/cluster-agent"]
