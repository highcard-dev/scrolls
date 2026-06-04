set -eu

cfg_dir="lgsm/config-lgsm/pzserver"
cfg_file="${cfg_dir}/pzserver.cfg"
mkdir -p "${cfg_dir}"

main_port="${DRUID_PORT_MAIN_1:-16261}"
udp_port="${DRUID_PORT_MAIN2_1:-16262}"
admin_password="${PZ_ADMIN_PASSWORD:-}"

if [ -z "${admin_password}" ]; then
	admin_password="$(od -An -N16 -tx1 /dev/urandom | tr -d ' \n')"
fi

cat > "${cfg_file}" <<EOF
startparameters="-servername \${selfname} -adminpassword ${admin_password} -port ${main_port} -udpport ${udp_port}"
EOF

chmod 600 "${cfg_file}"
