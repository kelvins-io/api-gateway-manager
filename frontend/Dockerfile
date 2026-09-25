FROM node:22-bookworm-slim AS builder

WORKDIR /app

COPY package.json package-lock.json ./
# vite 8 / rolldown 依赖平台原生包，需按构建机架构解析 optional deps
RUN npm install

COPY . .
RUN npm run build

FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 \
  CMD wget -qO- http://127.0.0.1/ || exit 1
