# ── dev stage ─────────────────────────────────────────────────────────────────
FROM node:22-alpine AS dev

RUN corepack enable pnpm

WORKDIR /app
COPY package.json pnpm-lock.yaml* ./
RUN pnpm install

EXPOSE 5173
CMD ["pnpm", "dev", "--host", "0.0.0.0"]

# ── build stage ───────────────────────────────────────────────────────────────
FROM node:22-alpine AS build

RUN corepack enable pnpm

WORKDIR /app
COPY package.json pnpm-lock.yaml* ./
RUN pnpm install --frozen-lockfile

COPY . .
RUN pnpm build

# ── prod stage: nginx serves static files ─────────────────────────────────────
FROM nginx:1.27-alpine AS prod

COPY --from=build /app/dist /usr/share/nginx/html

RUN printf 'server {\n\
    listen 80;\n\
    root /usr/share/nginx/html;\n\
    index index.html;\n\
    location / { try_files $uri $uri/ /index.html; }\n\
    location ~* \\.(js|css|woff2?|png|jpg|svg|ico)$ {\n\
        expires 1y;\n\
        add_header Cache-Control "public, immutable";\n\
    }\n\
}\n' > /etc/nginx/conf.d/default.conf

EXPOSE 80
