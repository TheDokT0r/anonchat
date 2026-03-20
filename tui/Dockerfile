FROM golang:1.23-alpine AS build
ENV GOTOOLCHAIN=local
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /anonchat-tui .

FROM alpine:3.19
COPY --from=build /anonchat-tui /usr/local/bin/anonchat-tui
ENTRYPOINT ["anonchat-tui"]
