#!/bin/sh
set -eu

. /opt/agm/normalize-base-path.sh

base=$(normalize_base_path "${VITE_BASE_PATH:-}")
if [ "$base" = "/" ]; then
  exit 0
fi

noslash=${base%/}

cat > /etc/nginx/conf.d/default.conf <<EOF
server {
    listen 80;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    # Docker 把宿主 5373 映到容器 80 时，绝对跳转会按监听端口丢掉宿主端口，
    # 浏览器落到 :80（Kong）而不是前端。改用相对 Location，由浏览器保留当前 host:port。
    absolute_redirect off;

    client_max_body_size 20m;

    location /api/ {
        proxy_pass http://backend:10000;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_connect_timeout 10s;
        proxy_read_timeout 60s;
    }

    location /health {
        proxy_pass http://backend:10000/health;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
    }

    location = / {
        return 302 ${base};
    }

    location = ${noslash} {
        return 301 ${base};
    }

    location = ${base} {
        root /usr/share/nginx/html;
        try_files /index.html =404;
    }

    location ${base} {
        rewrite ^${noslash}/(.*)$ /\$1 break;
        root /usr/share/nginx/html;
        try_files \$uri \$uri/ @spa;
    }

    location @spa {
        root /usr/share/nginx/html;
        try_files /index.html =404;
    }
}
EOF
