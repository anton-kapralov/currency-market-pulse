FROM ubuntu:latest
LABEL authors="Anton Kapralov"

WORKDIR /app

COPY target/consumer_linux-arm64 ./consumer

EXPOSE 8081

ENTRYPOINT ["/app/consumer"]
