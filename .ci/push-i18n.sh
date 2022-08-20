for dir in scrolls/*/.meta
do
    base=$(dirname $dir)
    cat=$(basename $base)
    mc cp --recursive ./$dir/* scrolls/druid-scrolls-staging/info/cat/$cat/
done

for dir in scrolls/*/*/.meta
do
    base=$(dirname $dir)
    scroll=$(basename $base)
    mc cp --recursive ./$dir/* scrolls/druid-scrolls-staging/info/variant/$scroll/
done