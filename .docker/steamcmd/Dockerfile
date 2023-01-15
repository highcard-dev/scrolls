FROM highcard/druidd-base:latest as base
FROM ubuntu:20.04

RUN dpkg --add-architecture i386
RUN apt update
RUN apt install -y lib32gcc1 ca-certificates wget curl libgcc1 vim net-tools jq moreutils
RUN wget https://github.com/mikefarah/yq/releases/download/v4.30.6/yq_linux_amd64 -O /usr/bin/yq
RUN chmod +x /usr/bin/yq

#RUN wget "https://github.com/highcard-dev/druidd/releases/download/latest/druid_linux_386.deb" -O "druidd.deb"
#RUN apt install ./druidd.deb
RUN useradd -ms /bin/bash druid
RUN mkdir /app
RUN chown druid:druid /app
USER druid

COPY --chown=druid:druid .docker/steamcmd/steamcmd /usr/bin/steamcmd
RUN chmod +x /usr/bin/steamcmd

COPY --chown=druid:druid .docker/prepare-switch-scroll.sh /usr/bin/prepare-switch-scroll
RUN chmod +x /usr/bin/prepare-switch-scroll

WORKDIR /home/druid
RUN mkdir -p /home/druid/.steam/steamcmd
RUN wget "https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz" -O /home/druid/.steam/steamcmd/steamcmd_linux.tar.gz
RUN cd /home/druid/.steam/steamcmd/ && tar zxvf steamcmd_linux.tar.gz
WORKDIR /app

COPY --from=base /usr/bin/druid* /usr/bin

CMD "/app/druid"
ENTRYPOINT [ "/app/druid" ]