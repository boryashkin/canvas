# Local development

- `cd cmd/client-app docker run -v "$PWD":/usr/src/app -w /usr/src/app -ti node:20-alpine3.16 sh`
- `npm i` (for the first time)
- `npm run dev`

# Build
- `cd nextjs`
- `docker build . -f ../../../build/frontend/Dockerfile`
- `docker image ls`
- `docker tag your_image_id ghcr.io/boryashkin/canvas:latest`
