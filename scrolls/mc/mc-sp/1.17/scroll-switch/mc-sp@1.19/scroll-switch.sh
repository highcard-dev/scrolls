if ["$NEW_SCROLL_DIR" == ""]; then
    NEW_SCROLL_DIR="./.new-scroll"
fi
if ["$SCROLL_DIR" == ""]; then
    SCROLL_DIR="./.scroll"
fi

cp $NEW_SCROLL_DIR/scroll.json $SCROLL_DIR/scroll.json

wget -O spigot-new.jar https://download.getbukkit.org/spigot/spigot-1.19.jar
rm spigot.jar
mv spigot-new.jar spigot.jar