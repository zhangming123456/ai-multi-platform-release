FROM docker.m.daocloud.io/library/node:22-slim

WORKDIR /app

COPY frontend/package.json ./
RUN npm install

COPY frontend/ .

EXPOSE 5500
CMD ["npm", "run", "dev", "--", "--host"]
