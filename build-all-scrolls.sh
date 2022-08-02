for d in scrolls/*/ ; do
    b=$(basename $d)
    tar -czvf scrolls/$b.tar.gz -C scrolls/$b .
done
