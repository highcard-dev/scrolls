#!/bin/sh
CURRENT_SCROLL=$(cat $SCROLL_DIR/scroll.yaml | yq .name)

echo "Current scroll: $CURRENT_SCROLL"

if [ "$CURRENT_SCROLL" != "$SCROLL" ]; then
    druid run switch-scroll $SCROLL
    rm $SCROLL_DIR/scroll-lock.json
else
    echo "Already on $SCROLL. Nothing to do."
fi