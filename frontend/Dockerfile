# STAGE 1: Build
FROM node:20-alpine AS build
WORKDIR /app

# Point to the subfolder where package.json actually lives
COPY weather-app/package*.json ./
RUN npm ci

# Copy the rest of the weather-app source code
COPY weather-app/ .
RUN npm run build --configuration=production

# STAGE 2: Serve
FROM nginx:alpine

# Clean out the default Nginx index file
RUN rm -rf /usr/share/nginx/html/*

COPY --from=build /app/dist/weather-app/ /usr/share/nginx/html/

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]