# syntax=docker/dockerfile:1
ARG GO_IMAGE=golang:1.25-alpine
ARG NODE_IMAGE=node:22-alpine
FROM ${GO_IMAGE} AS go-build
WORKDIR /src
COPY services/core/go.mod services/core/go.sum ./
RUN go mod download
COPY services/core/ ./
RUN CGO_ENABLED=0 go build -buildvcs=false -trimpath -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -buildvcs=false -trimpath -o /out/pdf-extractor ./cmd/pdf-extractor

FROM alpine:3.24 AS api
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 10001 repet && adduser -D -u 10001 -G repet repet && \
    mkdir -p /private/objects && chown -R repet:repet /private && chmod 700 /private/objects
WORKDIR /app
COPY --from=go-build /out/api ./api
COPY services/core/migrations/ ./migrations/
USER 10001:10001
EXPOSE 8080 8081
CMD ["/app/api"]

FROM alpine:3.24 AS pdf-extractor
RUN apk add --no-cache poppler-utils font-dejavu && \
    addgroup -g 10001 worker && adduser -D -u 10001 -G worker worker
COPY --from=go-build /out/pdf-extractor /usr/local/bin/pdf-extractor
USER 10001:10001
EXPOSE 8092
CMD ["/usr/local/bin/pdf-extractor"]

FROM ${NODE_IMAGE} AS web-build
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1
RUN corepack enable
COPY apps/web/package.json apps/web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --ignore-scripts
COPY apps/web/ ./
ARG HTML_PREVIEW_PUBLIC_URL
ENV API_INTERNAL_URL=http://api:8080 HTML_PREVIEW_PUBLIC_URL=$HTML_PREVIEW_PUBLIC_URL
RUN mkdir -p public && pnpm exec next build --webpack

FROM ${NODE_IMAGE} AS web
WORKDIR /app
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 HOSTNAME=0.0.0.0 PORT=3000
COPY --from=web-build --chown=node:node /app/.next/standalone ./
COPY --from=web-build --chown=node:node /app/.next/static ./.next/static
COPY --from=web-build --chown=node:node /app/public ./public
COPY --from=web-build --chown=node:node /app/lesson-content ./lesson-content
USER node
EXPOSE 3000
CMD ["node", "server.js"]
