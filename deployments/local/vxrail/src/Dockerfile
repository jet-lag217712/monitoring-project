# Build
FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/collector ./cmd/collector
RUN printf 'root:x:0:0:root:/root:/sbin/nologin\nnonroot:x:65532:65532:nonroot:/:/sbin/nologin\n' > /out/passwd \
    && printf 'root:x:0:\nnonroot:x:65532:\n' > /out/group

# Runtime
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/collector /collector
COPY configs/collector.example.yaml /configs/collector.example.yaml
COPY --from=build /out/passwd /etc/passwd
COPY --from=build /out/group /etc/group
USER nonroot
EXPOSE 9090
ENTRYPOINT ["/collector"]
CMD ["-config", "/configs/collector.example.yaml"]
