FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG LDFLAGS=""

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-} \
    go build -ldflags "${LDFLAGS}" -o /out/runtz-mcp ./cmd/runtz-mcp

FROM alpine:3.22

RUN adduser -D -g "" runtz
USER runtz

COPY --from=build /out/runtz-mcp /usr/local/bin/runtz-mcp

# Hosted deployments serve the streamable HTTP transport (docs-only). The Helm
# chart sets RUNTZ_MCP_HTTP=:8080; stdio mode is used when it is unset.
EXPOSE 8080

ENTRYPOINT ["runtz-mcp"]
