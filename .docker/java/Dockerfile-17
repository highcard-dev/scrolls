#Dockerhub does not allow for setting of build args :/
ARG IMAGE=openjdk:17-slim-bullseye
FROM highcard/druidd-base:latest as base
FROM $IMAGE
WORKDIR /app

RUN apt update
RUN apt install -y ca-certificates wget jq moreutils htop procps nano net-tools
RUN wget https://github.com/mikefarah/yq/releases/download/v4.30.6/yq_linux_amd64 -O /usr/bin/yq
RUN chmod +x /usr/bin/yq

#RUN wget "https://github.com/highcard-dev/druidd/releases/download/latest/druid_linux_386.deb" -O "druidd.deb"
#RUN apt install ./druidd.deb

RUN useradd -ms /bin/bash druid
USER druid

COPY --from=base /usr/bin/druid* /usr/bin
COPY --from=base /entrypoint.sh /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]