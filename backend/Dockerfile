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

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

EXPOSE 3000

ENTRYPOINT ["entrypoint.sh"]

CMD ["npm", "run", "start"]