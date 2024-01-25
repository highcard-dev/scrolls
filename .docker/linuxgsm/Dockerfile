ARG SHORTNAME
FROM highcard/druidd-base:latest as base
FROM gameservermanagers/gameserver:${SHORTNAME}

COPY --from=base /usr/bin/druid* /usr/bin
COPY --from=base /entrypoint.sh /entrypoint.sh

ENV LGSM_SERVERFILES=/app/resources/deployment/serverfiles
ENV LGSM_CONFIG=/app/resources/deployment/config-lgsm
ENV LGSM_LOGDIR=/app/resources/deployment/log
ENV LGSM_DATADIR=/app/resources/deployment/data


RUN useradd -ms /bin/bash druid
RUN chown druid:druid -R /app
USER druid

ENTRYPOINT [ "/entrypoint.sh" ]