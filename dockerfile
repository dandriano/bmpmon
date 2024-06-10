FROM golang:alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -o bmpw

FROM busybox:stable
WORKDIR /bin
COPY --from=builder /src/bmpmon .
ENTRYPOINT [ "/bin/bmpmon" ]