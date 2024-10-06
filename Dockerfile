FROM golang:latest AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./
RUN CGO_ENABLED=0 go build -o /out

FROM debian:latest AS ffmpeg-stage

RUN apt-get update && apt-get install -y curl xz-utils
RUN curl -L -O https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz
RUN tar -xf ffmpeg-release-amd64-static.tar.xz && cd ffmpeg-* && mv ffmpeg /

FROM gcr.io/distroless/static:latest AS release-stage

COPY --from=build-stage /out /out
COPY --from=ffmpeg-stage /ffmpeg /ffmpeg

ENTRYPOINT ["./out"]
