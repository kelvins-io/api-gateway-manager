FROM node:22-bookworm-slim AS builder

WORKDIR /app

COPY package.json package-lock.json ./
# vite 8 / rolldown 依赖平台原生包，需按构建机架构解析 optional deps
RUN npm install

COPY . .
# 页面访问前缀，例如 /proxy-path。空则挂在站点根路径。改前缀需要重新构建。
ARG VITE_BASE_PATH=
ENV VITE_BASE_PATH=$VITE_BASE_PATH
RUN npm run build

FROM nginx:1.27-alpine

ARG VITE_BASE_PATH=
ENV VITE_BASE_PATH=$VITE_BASE_PATH

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY docker/normalize-base-path.sh /opt/agm/normalize-base-path.sh
COPY docker/40-base-path.sh /docker-entrypoint.d/40-base-path.sh
COPY docker/healthcheck.sh /opt/agm/healthcheck.sh
RUN chmod +x /docker-entrypoint.d/40-base-path.sh /opt/agm/healthcheck.sh
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 \
  CMD /opt/agm/healthcheck.sh
