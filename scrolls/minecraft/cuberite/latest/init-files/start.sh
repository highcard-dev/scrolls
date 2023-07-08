
#!/bin/bash

MAX=${DRUID_MAX_MEMORY%?}
if [ -z "${MAX}" ];
then
    MAX=1024M
fi

./Cuberite