FROM golang:1.26.1-alpine@sha256:2389ebfa5b7f43eeafbd6be0c3700cc46690ef842ad962f6c5bd6be49ed82039 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG GO_PACKAGE=.
RUN CGO_ENABLED=0 go build -trimpath -o /out/backend ${GO_PACKAGE}

FROM alpine:3.23@sha256:ff7f0c727dd796aa36787bfcd9a64bf35a6ce72a73451b3359fde849b6703d63
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 app \
    && mkdir -p /app/upload /app/storage \
    && chown -R app:app /app
WORKDIR /app
COPY --from=build /out/backend /usr/local/bin/backend
USER app
EXPOSE 3000
CMD ["backend"]
