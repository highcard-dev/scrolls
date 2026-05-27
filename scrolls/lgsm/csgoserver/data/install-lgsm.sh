set -e

wget https://codeload.github.com/GameServerManagers/LinuxGSM/tar.gz/refs/heads/master -O lgsm.tar.gz
tar -xzf lgsm.tar.gz

#recreate lgsm dir
#rm -rf lgsm
mkdir -p lgsm


mv LinuxGSM-master/linuxgsm.sh .
mv LinuxGSM-master/lgsm/modules/ lgsm
chmod -R +x lgsm/modules/

rm -rf LinuxGSM-master
rm lgsm.tar.gz
