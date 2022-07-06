FROM alpine:latest
RUN apk update \
    && apk add tzdata \
    && apk add --no-cache bash \
    && apk add curl \
    && apk add iptables \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

WORKDIR /data

RUN chmod -R a+rw /data

COPY polaris-sidecar /data/polaris-sidecar

RUN chmod +x /data/polaris-sidecar
