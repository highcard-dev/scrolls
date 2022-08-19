for dir in scrolls/*/info
do
    base=$(dirname $dir)
    cat=$(basename $base)
    echo $cat $dir
    mc cp --recursive ./$dir/* scrolls/druid-scrolls-staging/info/cat/$cat/
done

for dir in scrolls/*/*/info
do
    base=$(dirname $dir)
    scroll=$(basename $base)
    echo $scroll $dir

    mc cp --recursive ./$dir/* scrolls/druid-scrolls-staging/info/scroll/$scroll/
done