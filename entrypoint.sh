#!/bin/bash
params=()
[[ $LOG_LEVEL ]] && params+=(-LOG_LEVEL $LOG_LEVEL)
[[ $TIME ]] && params+=(-TIME $TIME)
[[ $MAX_RETENTION ]] && params+=(-MAX_RETENTION $MAX_RETENTION)
[[ $BACKUP_SIZE_WARNING ]] && params+=(-BACKUP_SIZE_WARNING $BACKUP_SIZE_WARNING)
[[ $PORT ]] && params+=(-PORT $PORT)
[[ $X_API_KEY ]] && params+=(-X_API_KEY $X_API_KEY)

/app/goup ${params[@]}