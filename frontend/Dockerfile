# syntax=docker/dockerfile:1

FROM node:22-alpine AS build
WORKDIR /app

COPY package.json ./
RUN npm install

COPY index.html vite.config.js ./
COPY assets ./assets
COPY src ./src

ARG VITE_API_BASE_URL=/api
ARG VITE_AUTH_MODE=google
ARG VITE_GOOGLE_CLIENT_ID=
ARG VITE_DEMO_ENABLED=false
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL \
    VITE_AUTH_MODE=$VITE_AUTH_MODE \
    VITE_GOOGLE_CLIENT_ID=$VITE_GOOGLE_CLIENT_ID \
    VITE_DEMO_ENABLED=$VITE_DEMO_ENABLED

RUN npm run build

FROM nginx:1.27-alpine AS final
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
