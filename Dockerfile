FROM golang:1.26-alpine AS builder
COPY core/ /build/core/
COPY executor-shell/ /build/executor-shell/
WORKDIR /build/executor-shell
RUN go mod download
RUN CGO_ENABLED=0 go build -o /executor-shell ./cmd/module

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /executor-shell /usr/local/bin/executor-shell
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/executor-shell"]
