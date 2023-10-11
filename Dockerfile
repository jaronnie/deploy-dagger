FROM alpine:latest

COPY ./deploy-dagger /app/deploy-dagger
WORKDIR /app

ENTRYPOINT [ "./deploy-dagger", "server"]