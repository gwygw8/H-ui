FROM golang:1.20.14-alpine as builder

WORKDIR /app

ENV GOPROXY=https://goproxy.cn,direct

COPY . .

RUN go build -trimpath -ldflags -o h-ui -ldflags="-s -w"

FROM 1.20.14-alpine

WORKDIR /app

ENV GOPROXY=https://goproxy.cn,direct
ENV TZ=Asia/Shanghai
ENV GIN_MODE=release

COPY --from=builder /app/h-ui .

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk update \
    && apk add --no-cache bash tzdata ca-certificates \
    && rm -rf /var/cache/apk/* \
    && ln -snf /usr/share/zoneinfo/$TZ /etc/localtime \
    && echo $TZ > /etc/timezone

CMD ["./h-ui"]