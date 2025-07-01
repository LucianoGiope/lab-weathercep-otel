FROM golang:1.23.4 AS base

ARG PATH_API
ARG API_PORT

ENV PATH_API=${PATH_API}
ENV API_PORT=${API_PORT}
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

FROM base AS builder
WORKDIR /build
COPY . .
RUN go build ${PATH_API}/cmd/main.go && \
    chmod +x main

FROM alpine AS upx
RUN apk add --no-cache upx
COPY --from=builder /build/main /upx/main
RUN upx --best --lzma /upx/main -o /upx/main_compressed


FROM scratch AS main
WORKDIR /app
COPY --from=upx /upx/main_compressed /app/main
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT [ "./main" ]
EXPOSE ${API_PORT}
