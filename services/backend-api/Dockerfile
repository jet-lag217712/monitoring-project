# Build
FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

# Runtime
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/api /api
COPY configs/api.example.yaml /configs/api.example.yaml
USER nonroot:nonroot
EXPOSE 8000 9092
ENTRYPOINT ["/api"]
CMD ["-config", "/configs/api.example.yaml"]
