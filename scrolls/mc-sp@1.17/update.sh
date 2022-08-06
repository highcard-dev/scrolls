#default update script

SCRIPTPATH="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"

if [ ! -f "$SCRIPTPATH/scroll-lock.json" ]; then
	echo "Scroll lock not found. Skipping update"
	exit 0
fi

versionsDirs=$(find update/* -maxdepth 0 -type d | sort --version-sort)
current=$(cat $SCRIPTPATH/scroll-lock.json | jq -r .version)

for versionsDir in $versionsDirs
do
	version=$(basename $versionsDir)
	if [ ! "$(printf '%s\n' "$version" "$current" | sort -V | head -n1)" = "$version" ] ;
	then
		if [ -f "$version/update.sh" ]; then
			sh $version/update.sh
		else
			echo "Warning: update $version has no update.sh... skipping"
		fi
	fi
done