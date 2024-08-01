FROM ubuntu:20.04

RUN dpkg --add-architecture i386
RUN apt update
RUN apt install -y lib32gcc1 ca-certificates wget curl libgcc1 vim net-tools jq moreutils net-tools zip
RUN wget https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 -O /usr/bin/yq
RUN chmod +x /usr/bin/yq

ARG CACHEBUST=1 
RUN wget -O /usr/bin/druid https://github.com/highcard-dev/druid-cli/releases/latest/download/druid
RUN wget -O /usr/bin/druid_rcon https://github.com/highcard-dev/druid-cli/releases/latest/download/druid_rcon
RUN wget -O /usr/bin/druid_rcon_web_rust https://github.com/highcard-dev/druid-cli/releases/latest/download/druid_rcon_web_rust
RUN wget -O /entrypoint.sh https://github.com/highcard-dev/druid-cli/releases/latest/download/entrypoint.sh

RUN chmod +x /usr/bin/druid
RUN chmod +x /usr/bin/druid_rcon
RUN chmod +x /usr/bin/druid_rcon_web_rust
RUN chmod +x /entrypoint.sh

#RUN wget "https://github.com/highcard-dev/druidd/releases/download/latest/druid_linux_386.deb" -O "druidd.deb"
#RUN apt install ./druidd.deb
RUN useradd -ms /bin/bash druid
RUN mkdir /app
RUN chown druid:druid /app
USER druid

COPY --chown=druid:druid .docker/steamcmd/steamcmd /usr/bin/steamcmd
RUN chmod +x /usr/bin/steamcmd

WORKDIR /home/druid
RUN mkdir -p /home/druid/.steam/steamcmd
RUN wget "https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz" -O /home/druid/.steam/steamcmd/steamcmd_linux.tar.gz
RUN cd /home/druid/.steam/steamcmd/ && tar zxvf steamcmd_linux.tar.gz
WORKDIR /app


ENTRYPOINT [ "/entrypoint.sh" ]