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

# Always serves the streamable HTTP transport (docs-only). The Helm chart
# sets RUNTZ_MCP_ADDR=:8080.
EXPOSE 8080

ENTRYPOINT ["runtz-mcp"]
