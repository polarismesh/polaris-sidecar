FROM alpine:latest

RUN sed -i 's!http://dl-cdn.alpinelinux.org/!https://mirrors.tencent.com/!g' /etc/apk/repositories

RUN set -eux && \
    apk add openjdk8 && \
    apk add bind-tools && \
    apk add busybox-extras && \
    apk add findutils && \
    apk add tcpdump && \
    apk add tzdata && \
    apk add curl && \
    apk add bash && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    date

WORKDIR /data

RUN chmod -R a+rw /data

COPY polaris-sidecar /data/polaris-sidecar
COPY polaris.yaml /data/polaris.yaml

RUN chmod +x /data/polaris-sidecar

ENTRYPOINT ["polaris-sidecar", "start"]