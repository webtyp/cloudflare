# A Cloudflare Workers runtime for the tests: Miniflare (workerd) with a real, local D1 database.
# glibc image on purpose: the workerd binary does not run on Alpine.
FROM node:22-slim
WORKDIR /runner
RUN npm install --no-save --no-audit --no-fund miniflare@4.20260730.0
COPY run.cjs ./
ENTRYPOINT ["node", "run.cjs"]
