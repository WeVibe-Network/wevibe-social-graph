FROM golang:1.26-alpine AS builder

WORKDIR /build

RUN apk add --no-cache build-base

COPY Echo-Internal/wevibe-social-graph/go.mod Echo-Internal/wevibe-social-graph/go.sum ./
RUN go mod download

COPY Echo-Internal/wevibe-social-graph/ ./

RUN CGO_ENABLED=1 GOOS=linux go build -o wevibe-social-graph ./cmd/server

FROM alpine:3.19

RUN apk add --no-cache ca-certificates sqlite-libs

COPY --from=builder /build/wevibe-social-graph /usr/local/bin/wevibe-social-graph

ENV SOCIAL_GRAPH_DB_PATH=/data/social-graph.db
EXPOSE 4470

CMD ["wevibe-social-graph"]
