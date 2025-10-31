FROM golang:1.25.3-alpine AS build
WORKDIR /src
COPY services/core ./services/core
COPY shared ./shared
COPY services/authentication ./services/authentication
WORKDIR /src/services/authentication
RUN go mod tidy && go build -o /bin/authentication ./cmd

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /bin/authentication /usr/local/bin/authentication
CMD ["authentication"]
