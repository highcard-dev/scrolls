#!/bin/bash
set -e
source "$HOME/.sdkman/bin/sdkman-init.sh"


wget https://hub.spigotmc.org/jenkins/job/BuildTools/lastSuccessfulBuild/artifact/target/BuildTools.jar
export PAGER=cat
PAGER=cat

sdk install java 20-open || true
sdk install java 18-open || true
sdk install java 17-open || true
sdk install java 16-open || true

sdk use java 20-open -y
java -jar BuildTools.jar --rev 1.20.1
java -jar BuildTools.jar --rev 1.20


sdk use java 18-open -y
java -jar BuildTools.jar --rev 1.19.4
java -jar BuildTools.jar --rev 1.19.3
java -jar BuildTools.jar --rev 1.19.2
java -jar BuildTools.jar --rev 1.19.1
java -jar BuildTools.jar --rev 1.19

java -jar BuildTools.jar --rev 1.18.2


sdk use java 17-open -y
java -jar BuildTools.jar --rev 1.18.1
java -jar BuildTools.jar --rev 1.18


sdk use java 16-open -y
java -jar BuildTools.jar --rev 1.17.1
java -jar BuildTools.jar --rev 1.17