
### Build viewscreen binary
FROM golang:latest AS builder
RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*

RUN go get -u github.com/gobuffalo/packr/v2/packr2

WORKDIR /go/src/github.com/xenking/viewscreen

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY static ./static
COPY templates ./templates
COPY viewscreen ./

ENV GODEBUG="netdns=go http2server=0" GOPATH="/go" GO111MODULE=on

RUN packr2 \
    && go fmt \
    && go vet --all

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -v --compiler gc --ldflags "-extldflags -static -s -w -X main.version=$(git describe --tags)" -o /usr/local/bin/viewscreen \
    && packr2 clean


### Build an alpine image with binary
FROM alpine:latest
ENV MUSL_LOCPATH="/usr/share/i18n/locales/musl"
RUN apk --no-cache add \
    libintl \
    curl \
    ffmpeg \
    wget \
    x264 \
    file \
    imagemagick \
    enca && \
	apk --no-cache --virtual .locale_build add cmake make musl-dev gcc gettext-dev git && \
	git clone https://gitlab.com/rilian-la-te/musl-locales && \
	cd musl-locales && cmake -DLOCALE_PROFILE=OFF -DCMAKE_INSTALL_PREFIX:PATH=/usr . && make && make install && \
	cd .. && rm -r musl-locales && \
	apk del .locale_build

WORKDIR /data

ENV LC_ALL=ru_RU.UTF-8
COPY --from=builder /usr/local/bin/viewscreen /usr/local/bin/viewscreen
COPY --from=builder /go/src/github.com/xenking/viewscreen/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/viewscreen /usr/local/bin/entrypoint.sh

ENTRYPOINT [ "/usr/local/bin/entrypoint.sh" ]