#nice command but useless
#changedScrolls=$(git diff --dirstat=files,0,cumulative HEAD~1 scrolls/ | awk '{print $2}' | cut -d'/' -f2 | uniq | grep -v -e '^$')


if [ -f latest.json ]; then
    mv latest.json old-latest.json
else
    echo "{}" > old-latest.json
fi

echo "{}" > latest.json

for d in scrolls/*/ ; do
    ref=$(git log --format="%H" -n 1 -- $d)
    scrollName=$(echo "$d" | cut -d'/' -f2)
    jq '."'"$scrollName"'" = "'"$ref"'"' latest.json|sponge latest.json
done

echo "scrolls that changed"
changedScrolls=$(jd -f patch latest.json old-latest.json | jq -r '.[].path' | uniq )

ref=$(git rev-parse HEAD)
for s in $changedScrolls ; do
    echo $s
    jq '.ref = "'"$ref"'"' scrolls$s/scroll.json|sponge scrolls$s/scroll.json
    tar -czvf scrolls$s.tar.gz -C scrolls$s .
done