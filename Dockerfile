### First stage ###
FROM golang:alpine AS builder
WORKDIR /app
COPY go.sum go.mod ./
RUN go mod download
COPY . ./
RUN go build -o /beko-scraper .

### Second stage ###
FROM alpine
WORKDIR /app
COPY --from=builder /beko-scraper /beko-scraper
COPY --from=builder /app/output /app/output
ENTRYPOINT ["/beko-scraper"]