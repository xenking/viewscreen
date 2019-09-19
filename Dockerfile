FROM alpine:latest

RUN apk --no-cache add \
    curl \
    ffmpeg \
    wget \
    x264

WORKDIR /data

COPY viewscreen-linux-amd64 /usr/bin/viewscreen
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN ["chmod", "+x", "/usr/local/bin/entrypoint.sh"]

#ENTRYPOINT ["/usr/bin/viewscreen"]
ENTRYPOINT [ "/usr/local/bin/entrypoint.sh" ]