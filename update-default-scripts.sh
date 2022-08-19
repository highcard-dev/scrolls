SCROLLS_DIRS=$(ls -d scrolls/*/*/)

#scroll switch
for SCROLL in $SCROLLS_DIRS
do
    if [ ! -d "${SCROLL}scroll-switch" ]; then
        echo "[scroll-switch] skipping $SCROLL"
        continue
    fi

    SCROLL_SWITCH_DIRS=$(ls -d ${SCROLL}scroll-switch/*/)
    for SCROLL_SWITCH in $SCROLL_SWITCH_DIRS
    do  
        SCRIPT_PATH=${SCROLL_SWITCH}scroll-switch.sh
        if [ ! -f "$SCRIPT_PATH" ]; then
            echo "[scroll-switch] WARNING $SCRIPT_PATH does not exist, but it should"
            continue
        fi
        FIRST_LINE=$(head -n 1 $SCRIPT_PATH)
        if [[ "$FIRST_LINE" = "#default scroll switch script" ]]; then
            echo "[scroll-switch] found default scroll switch script. Updating ($SCRIPT_PATH)"
            cp ./common/default-scroll-switch.sh $SCRIPT_PATH
        else
            echo "$FIRST_LINE"
        fi
    done
done

#scroll updates
for SCROLL in $SCROLLS_DIRS
do
    SCRIPT_PATH=${SCROLL}update.sh
    if [ ! -f "$SCRIPT_PATH" ]; then
        echo "[update] skipping $SCROLL"
        continue
    fi
    FIRST_LINE=$(head -n 1 $SCRIPT_PATH)
    if [[ "$FIRST_LINE" = "#default update script" ]]; then
        echo "[update] found default update script... updating ($SCRIPT_PATH)"
        cp ./common/default-update.sh $SCRIPT_PATH
    fi
done