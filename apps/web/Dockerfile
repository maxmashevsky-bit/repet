FROM node:22-alpine AS build
WORKDIR /app
RUN corepack enable
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --ignore-scripts
COPY . .
ENV API_INTERNAL_URL=http://api:8080
RUN pnpm build

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production
ENV API_INTERNAL_URL=http://api:8080
COPY --from=build /app ./
EXPOSE 3000
CMD ["node", "node_modules/next/dist/bin/next", "start", "-H", "0.0.0.0"]
