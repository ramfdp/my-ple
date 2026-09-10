FROM golang:1.26.1-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG GO_PACKAGE=.
RUN CGO_ENABLED=0 go build -trimpath -o /out/backend ${GO_PACKAGE}

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 app \
    && mkdir -p /app/upload /app/storage \
    && chown -R app:app /app
WORKDIR /app
COPY --from=build /out/backend /usr/local/bin/backend
USER app
EXPOSE 3000
CMD ["backend"]
