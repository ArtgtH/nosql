FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/service ./cmd


FROM gcr.io/distroless/static-debian12:nonroot as runtime

WORKDIR /app
COPY --from=build /out/service /app/service

USER nonroot:nonroot

ENTRYPOINT ["/app/service"]

EXPOSE 8080
