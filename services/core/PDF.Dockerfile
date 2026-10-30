FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /pdf-extractor ./cmd/pdf-extractor

FROM alpine:3.24
RUN apk add --no-cache poppler-utils font-dejavu && addgroup -g 10001 worker && adduser -D -u 10001 -G worker worker
COPY --from=build /pdf-extractor /usr/local/bin/pdf-extractor
USER 10001:10001
EXPOSE 8092
CMD ["/usr/local/bin/pdf-extractor"]
