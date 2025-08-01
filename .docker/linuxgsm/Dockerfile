ARG SHORTNAME
FROM gameservermanagers/gameserver:${SHORTNAME}

RUN apt-get update \
    && apt-get install -y \
    net-tools \
    psmisc \
    && apt-get -y autoremove \
    && apt-get -y clean \
    && rm -rf /var/lib/apt/lists/* \
    && rm -rf /tmp/* \
    && rm -rf /var/tmp/*

ARG CACHEBUST=1 
RUN wget -O /usr/bin/druid https://github.com/highcard-dev/druid-cli/releases/latest/download/druid
RUN wget -O /usr/bin/druid_rcon https://github.com/highcard-dev/druid-cli/releases/latest/download/druid_rcon
RUN wget -O /usr/bin/druid_rcon_web_rust https://github.com/highcard-dev/druid-cli/releases/latest/download/druid_rcon_web_rust
RUN wget -O /entrypoint.sh https://github.com/highcard-dev/druid-cli/releases/latest/download/entrypoint.sh

RUN chmod +x /usr/bin/druid
RUN chmod +x /usr/bin/druid_rcon
RUN chmod +x /usr/bin/druid_rcon_web_rust
RUN chmod +x /entrypoint.sh

RUN sudo usermod -d /app/resources/deployment linuxgsm

ENV LGSM_SERVERFILES=/app/resources/deployment/serverfiles
ENV LGSM_CONFIG=/app/resources/deployment/config-lgsm
ENV LGSM_LOGDIR=/app/resources/deployment/log
ENV LGSM_DATADIR=/app/resources/deployment/data


RUN useradd -ms /bin/bash druid
RUN chown druid:druid -R /app
USER druid

ENTRYPOINT [ "/entrypoint.sh" ]