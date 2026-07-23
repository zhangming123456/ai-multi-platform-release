FROM docker.m.daocloud.io/library/node:22-slim

WORKDIR /app

COPY frontend/package.json ./
RUN npm install

COPY frontend/ .

EXPOSE 5173
CMD ["npm", "run", "dev", "--", "--host"]
