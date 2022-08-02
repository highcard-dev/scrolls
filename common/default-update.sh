#default update script
versions=$(find * -maxdepth 0 -type d | sort --version-sort)
current=$(cat scroll-lock.json | jq -r .version)

SCRIPTPATH="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"

for version in $versions
do

	if [ ! "$(printf '%s\n' "$version" "$current" | sort -V | head -n1)" = "$version" ] ;
	then
		if [ -f "$version/update.sh" ]; then
			sh $version/update.sh
		else
			echo "Warning: update $version has no update.sh... skipping"
		fi
	fi
done