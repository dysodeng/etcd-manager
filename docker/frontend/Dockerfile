# ---- 构建阶段 ----
FROM node:22.14-alpine AS builder

WORKDIR /src
COPY web/package.json web/package-lock.json* ./
RUN npm install

COPY web/ .
RUN npm run build

# ---- 运行阶段 ----
FROM nginx:alpine

COPY docker/frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /src/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
