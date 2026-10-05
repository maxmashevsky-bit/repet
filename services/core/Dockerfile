FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /api ./cmd/api

FROM alpine:3.22
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /api /app/api
COPY --from=build /src/migrations /app/migrations
EXPOSE 8080
CMD ["/app/api"]
