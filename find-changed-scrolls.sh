#nice command but useless
#changedScrolls=$(git diff --dirstat=files,0,cumulative HEAD~1 scrolls/ | awk '{print $2}' | cut -d'/' -f2 | uniq | grep -v -e '^$')


if [ ! -f old-latest.json ]; then
    echo "creating old-latest.json"
    echo "{}" > old-latest.json
fi

echo "{}" > latest.json

for d in scrolls/*/ ; do
    version=$(cat $d/scroll.json | jq -r '.version')
    echo $version
    scrollName=$(echo "$d" | cut -d'/' -f2)
    jq '."'"$scrollName"'" = "'"$version"'"' latest.json|sponge latest.json
done

echo "scrolls that changed"
changedScrolls=$(node get-diff.js)

for s in $changedScrolls ; do
    echo $s
    tar -czvf scrolls/$s.tar.gz -C scrolls/$s .
done