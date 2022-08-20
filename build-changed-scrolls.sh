if [ ! -f old-latest.json ]; then
    echo "creating old-latest.json"
    echo "{}" > old-latest.json
fi

echo "{}" > latest.json

for d in scrolls/*/*/*/ ; do
    version=$(cat $d/scroll.json | jq -r '.version')
    echo $d
    scrollName=$(echo "$d" | cut -d'/' -f4)
    jq '."'"$scrollName"'" = "'"$version"'"' latest.json|sponge latest.json
done

echo "scrolls that changed"
changedScrolls=$(node get-diff.js)
echo $changedScrolls

for d in scrolls/*/*/*/ ; do
    b=$(basename $d)
    for  i in $changedScrolls ; do
        if [[ "$i" == "$b" ]]; then
            tar -czvf scrolls/$b.tar.gz -C $d .
        fi
    done
done