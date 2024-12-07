FROM oven/bun:edge as frontend-builder

WORKDIR /app

COPY package.json bun.lockb ./
RUN bun install

COPY . .

RUN bun run build

FROM caddy:latest as frontend

COPY --from=frontend-builder /app/dist /srv

EXPOSE 80

CMD ["caddy", "file-server", "--root", "/srv"]

