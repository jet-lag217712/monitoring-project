# Build
FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/collector ./cmd/collector

# Runtime
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/collector /collector
COPY configs/collector.example.yaml /configs/collector.example.yaml
USER nonroot:nonroot
EXPOSE 9090
ENTRYPOINT ["/collector"]
CMD ["-config", "/configs/collector.example.yaml"]
