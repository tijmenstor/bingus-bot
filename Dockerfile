FROM ubuntu:latest

WORKDIR /bingus

RUN apt update && apt install -y ffmpeg

COPY bingus-bot .
COPY commands.json .
COPY sounds ./sounds

CMD ["./bingus-bot"]