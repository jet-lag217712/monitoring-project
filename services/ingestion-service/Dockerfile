# Build
FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/ingestion ./cmd/ingestion

# Runtime
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/ingestion /ingestion
COPY configs/ingestion.example.yaml /configs/ingestion.example.yaml
USER nonroot:nonroot
EXPOSE 9091
ENTRYPOINT ["/ingestion"]
CMD ["-config", "/configs/ingestion.example.yaml"]
