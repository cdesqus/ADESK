FROM node:20-alpine

WORKDIR /app

# Update npm to latest version to avoid warnings
RUN npm install -g npm@latest

COPY package.json package-lock.json* ./
RUN npm install

COPY . .

EXPOSE 5173

CMD ["npm", "run", "dev", "--", "--host"]
