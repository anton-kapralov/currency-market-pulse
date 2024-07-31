FROM ubuntu:latest
LABEL authors="Anton Kapralov"

WORKDIR /app

COPY target/trends_linux-arm64 ./trends

EXPOSE 8082

ENTRYPOINT ["/app/trends"]
