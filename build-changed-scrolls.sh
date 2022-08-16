if [ ! -f old-latest.json ]; then
    echo "creating old-latest.json"
    echo "{}" > old-latest.json
fi

echo "{}" > latest.json

for d in scrolls/*/ ; do
    version=$(cat $d/scroll.json | jq -r '.version')
    echo $d
    scrollName=$(echo "$d" | cut -d'/' -f2)
    jq '."'"$scrollName"'" = "'"$version"'"' latest.json|sponge latest.json
done

echo "scrolls that changed"
changedScrolls=$(node get-diff.js)
echo $changedScrolls

for s in $changedScrolls ; do
    tar -czvf scrolls/$s.tar.gz -C scrolls/$s .
done