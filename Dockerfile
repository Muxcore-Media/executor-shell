FROM golang:1.26-alpine AS builder
COPY core/ /build/core/
COPY executor-shell/ /build/executor-shell/
WORKDIR /build/executor-shell
RUN go mod download
RUN CGO_ENABLED=0 go build -o /executor-shell ./cmd/module
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /executor-shell /
ENTRYPOINT ["/executor-shell"]
