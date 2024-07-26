FROM ubuntu:latest
LABEL authors="Anton Kapralov"

WORKDIR /app

COPY target/importer_linux-arm64 ./importer

ENTRYPOINT ["/app/importer"]
