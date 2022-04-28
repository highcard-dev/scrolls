if ls scrolls/*.tar.gz 1> /dev/null 2>&1; then
    mc cp --recursive ./scrolls/*.tar.gz scrolls/druid-scrolls-staging
    mc cp ./latest.json scrolls/druid-scrolls-staging
else
    echo "No scrolls to push"
fi