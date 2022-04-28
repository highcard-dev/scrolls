changedScrolls=$(git diff --dirstat=files,0,cumulative HEAD~1 scrolls/ | awk '{print $2}' | cut -d'/' -f2 | uniq | grep -v -e '^$')



for s in $changedScrolls
do 
    echo $s
    oldVersion=$(cat scrolls/$s/scroll.json | jq -r .version)
    cat latest.json | jq '.["$s"]'
done