set -e

rm -rf LinuxGSM-master
druid progress download --output lgsm.tar.gz --label "Downloading LinuxGSM" \
  https://codeload.github.com/GameServerManagers/LinuxGSM/tar.gz/refs/heads/master
tar -xzf lgsm.tar.gz

mkdir -p lgsm

rm -f linuxgsm.sh
rm -rf lgsm/modules
mv LinuxGSM-master/linuxgsm.sh .
mv LinuxGSM-master/lgsm/modules lgsm/modules
chmod -R +x lgsm/modules/

rm -rf LinuxGSM-master
rm lgsm.tar.gz
