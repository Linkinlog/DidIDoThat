FROM oven/bun:edge as frontend-builder

WORKDIR /app

COPY package.json bun.lockb ./
RUN bun install

COPY . .

RUN bun run build
