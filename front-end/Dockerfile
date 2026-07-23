# Front-end: Hardened Node.js Dockerfile
# Uses a minimal non-root multi-stage build to reduce attack surface

FROM node:20-alpine3.21 AS builder

WORKDIR /app

COPY package.json ./
RUN npm install --production

# Production stage - minimal image
FROM node:20-alpine3.21

# Security: run as non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy only necessary files
COPY --from=builder /app/node_modules ./node_modules
COPY package.json ./
COPY src/ ./src/
COPY views/ ./views/
COPY public/ ./public/

# Security: set non-root user
USER appuser

# Expose the application port
EXPOSE 8079

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8079/health || exit 1

CMD ["node", "src/app.js"]
