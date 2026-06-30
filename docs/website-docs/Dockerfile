# Stage 1 — deps
FROM node:22-alpine AS deps
RUN corepack enable pnpm
WORKDIR /app
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --no-frozen-lockfile --ignore-scripts

# Stage 2 — build
FROM node:22-alpine AS builder
RUN corepack enable pnpm
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN pnpm build

# Stage 3 — runner
FROM node:22-alpine AS runner
RUN corepack enable pnpm
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000

# Only copy what Next.js needs to run
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./package.json

EXPOSE 3000
CMD ["pnpm", "start"]
