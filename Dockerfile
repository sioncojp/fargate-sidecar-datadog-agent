FROM golang:latest AS build

ENV GO111MODULE auto

ADD . /src
WORKDIR /src
RUN make build

FROM datadog/agent:latest

COPY --from=build /src/bin/fargate-sidecar-datadog-agent .
RUN chmod +x fargate-sidecar-datadog-agent

CMD ["./fargate-sidecar-datadog-agent", "/init"]