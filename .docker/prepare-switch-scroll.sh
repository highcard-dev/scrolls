#!/bin/sh
CURRENT_SCROLL=$(cat $SCROLL_DIR/scroll.yaml | yq .name)

echo "Current scroll: $CURRENT_SCROLL"

DESIRED_SCROLL="$SCROLL@$SCROLL_VERSION"

if [ "$CURRENT_SCROLL" != "$DESIRED_SCROLL" ]; then
    druid run switch-scroll $DESIRED_SCROLL
    rm $SCROLL_DIR/scroll-lock.json
else
    echo "Already on $CURRENT_SCROLL. Nothing to do."
fi