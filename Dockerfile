FROM alpine:3.24.2

RUN sed -i 's!http://dl-cdn.alpinelinux.org/!https://mirrors.tencent.com/!g' /etc/apk/repositories

RUN set -eux && \
    apk upgrade --no-cache && \
    apk add --no-cache \
        bash \
        bind-tools \
        busybox-extras \
        curl \
        findutils \
        tcpdump \
        tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    date

WORKDIR /data

RUN chmod -R a+rw /data

COPY polaris-sidecar /data/polaris-sidecar

RUN chmod +x /data/polaris-sidecar

ENTRYPOINT ["/data/polaris-sidecar", "start"]
