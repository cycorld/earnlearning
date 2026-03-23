FROM node:20-alpine AS builder
WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci --legacy-peer-deps
COPY frontend/ .
COPY changelog/ public/changelog/
ARG VITE_API_URL=/api
ARG VITE_WS_URL=
ARG BUILD_NUMBER=dev
ARG COMMIT_SHA=local
ENV BUILD_NUMBER=$BUILD_NUMBER
ENV COMMIT_SHA=$COMMIT_SHA
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
