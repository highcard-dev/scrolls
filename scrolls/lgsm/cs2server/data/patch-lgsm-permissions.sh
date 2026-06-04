set -e

permissions_file="lgsm/modules/check_permissions.sh"

if [ ! -f "${permissions_file}" ]; then
  echo "LinuxGSM permissions module not found: ${permissions_file}" >&2
  exit 1
fi

if grep -q "Druid skips LinuxGSM ownership checks" "${permissions_file}"; then
  exit 0
fi

tmp_file="${permissions_file}.tmp"
sed 's/^[[:space:]]*fn_check_ownership$/\t# Druid skips LinuxGSM ownership checks because Kubernetes PVCs may remap file owners.\n\t# Permission and executable checks still run below./' "${permissions_file}" > "${tmp_file}"

if cmp -s "${permissions_file}" "${tmp_file}"; then
  rm -f "${tmp_file}"
  echo "LinuxGSM ownership check call not found in ${permissions_file}" >&2
  exit 1
fi

cat "${tmp_file}" > "${permissions_file}"
rm -f "${tmp_file}"
