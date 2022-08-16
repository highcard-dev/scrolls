for dir in ./scrolls/*.tar.gz
do
    filename=$(basename -- "$dir")
    filename="${filename%.tar.gz}"
    mc cp --recursive ./scrolls/$filename/info/ scrolls/druid-scrolls-staging/info/$filename/
done