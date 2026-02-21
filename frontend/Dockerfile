FROM node:lts-slim AS deps

WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm ci

FROM node:lts-slim AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

RUN npm run build

FROM node:lts-slim AS runner

RUN apt update && apt upgrade -y && \
    apt clean -y && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/server.js ./server.js

EXPOSE 4173

CMD ["npm", "run", "start"]