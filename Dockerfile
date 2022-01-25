FROM alpine:latest as base

RUN apk update && apk add ca-certificates curl nginx supervisor && rm -rf /var/cache/apk/*

COPY supervisord.conf /etc
COPY nginx.conf /etc/nginx
COPY nginx-default.conf /etc/nginx/conf.d/default.conf

# Copy AMD binaries
FROM base AS image-amd64-

COPY linux/amd64/fake-service /app/fake-service
RUN chmod +x /app/fake-service

# Copy Arm 6 binaries
FROM base AS image-arm-v6

COPY linux/arm6/fake-service /app/fake-service
RUN chmod +x /app/fake-service

# Copy Arm 7 binaries
FROM base AS image-arm-v7

COPY linux/arm7/fake-service /app/fake-service
RUN chmod +x /app/fake-service

# Copy Arm 8 binaries
FROM base AS image-arm64-

COPY linux/arm64/fake-service /app/fake-service
RUN chmod +x /app/fake-service

FROM image-${TARGETARCH}-${TARGETVARIANT}

ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ARG BUILDPLATFORM
ARG BUILDARCH

RUN echo "I am running on $BUILDPLATFORM, building for $TARGETPLATFORM $TARGETARCH $TARGETVARIANT"  

ENTRYPOINT ["supervisord"]
CMD ["--nodaemon", "--configuration", "/etc/supervisord.conf"]
