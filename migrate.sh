#!/bin/bash
set -e

banner() {
    echo "+------------------------------------------+"
    printf "| %-40s |\n" "$(date)"
    echo "|                                          |"
    printf "|$(tput bold) %-40s $(tput sgr0)|\n" "$@"
    echo "+------------------------------------------+"
}

echo " ____           _                _      __  __ _                 _   _              "
echo " |  _ \         | |              | |    |  \/  (_)               | | (_)            "
echo " | |_) | ___  __| |_ __ ___   ___| | __ | \  / |_  __ _ _ __ __ _| |_ _  ___  _ __  "
echo " |  _ < / _ \/ _  |  __/ _ \ / __| |/ / | |\/| | |/ _  | '__/ _  | __| |/ _ \| '_ \ "
echo " | |_) |  __/ (_| | | | (_) | (__|   <  | |  | | | (_| | | | (_| | |_| | (_) | | | |"
echo " |____/ \___|\__,_|_|  \___/ \___|_|\_\ |_|  |_|_|\__, |_|  \__,_|\__|_|\___/|_| |_|"
echo "                                                   __/ |                            "
echo "                                                  |___/                             "

LOG_DIR="migration-log"

if [[ -n "$1" ]]; then
    BEDROCK_START_BLOCK_NUM=$1
else
    echo "bedrock start block number not set."
fi

if [[ -n "$2" ]]; then
    GETH_DATA_DIR=$2
else
    echo "data directory not set."
    exit 1
fi

if [ ! -d "$LOG_DIR" ]; then
    mkdir "$LOG_DIR"
fi

ARTIFACT_PATH="/tmp/migration-artifact"
if [ ! -d "$ARTIFACT_PATH" ]; then
    mkdir "$ARTIFACT_PATH"
fi

banner "Init"
# boot up geth with new database to supply chain config
# this is only needed for new database
./build/bin/geth --datadir="$GETH_DATA_DIR" --http 2> "$LOG_DIR"/start.log &
PID=$!
sleep 10
curl --location 'localhost:8545/' \
--header 'Content-Type: application/json' \
--data '{
	"jsonrpc":"2.0",
	"method":"eth_blockNumber",
	"params":[],
	"id":83
}'
kill $PID
sleep 10

banner "Export Blocks"
time ./build/bin/geth --datadir="$GETH_DATA_DIR" export "$ARTIFACT_PATH"/blocks.rlp 0 "$BEDROCK_START_BLOCK_NUM" 2> "$LOG_DIR"/export_blocks.log

banner "Export Total Difficulty"
time ./build/bin/geth --datadir="$GETH_DATA_DIR" export-totaldifficulty "$ARTIFACT_PATH"/totaldifficulty.rlp 0 "$BEDROCK_START_BLOCK_NUM" 2> "$LOG_DIR"/export_totaldifficulty.log

banner "Export Receipts"
time ./build/bin/geth --datadir="$GETH_DATA_DIR" export-receipts "$ARTIFACT_PATH"/receipts.rlp 0 "$BEDROCK_START_BLOCK_NUM" 2> "$LOG_DIR"/export_receipts.log

banner "Export State"
time ./build/bin/geth --datadir="$GETH_DATA_DIR" dump --iterative "$BEDROCK_START_BLOCK_NUM" 1> "$ARTIFACT_PATH"/world_trie_state.jsonl 2> "$LOG_DIR"/export_state.log
