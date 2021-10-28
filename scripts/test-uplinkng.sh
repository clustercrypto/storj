#!/usr/bin/env bash
set -ueo pipefail

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source $SCRIPTDIR/utils.sh

TMPDIR=$(mktemp -d -t tmp.XXXXXXXXXX)

cleanup(){
    rm -rf "$TMPDIR"
    echo "cleaned up test successfully"
}
trap cleanup EXIT
trap 'failure ${LINENO} "$BASH_COMMAND"' ERR

BUCKET=bucket-123
SRC_DIR=$TMPDIR/source
DST_DIR=$TMPDIR/dst
UPLINK_DIR=$TMPDIR/uplink

mkdir -p "$SRC_DIR" "$DST_DIR"

random_bytes_file "2KiB"    "$SRC_DIR/small-upload-testfile"          # create 2KiB file of random bytes (inline)
random_bytes_file "5MiB"    "$SRC_DIR/big-upload-testfile"            # create 5MiB file of random bytes (remote)
# this is special case where we need to test at least one remote segment and inline segment of exact size 0
random_bytes_file "12MiB"   "$SRC_DIR/multisegment-upload-testfile"   # create 12MiB file of random bytes (1 remote segments + inline)
random_bytes_file "13MiB"   "$SRC_DIR/diff-size-segments"             # create 13MiB file of random bytes (2 remote segments)

random_bytes_file "100KiB"  "$SRC_DIR/put-file"                       # create 100KiB file of random bytes (remote)

export STORJ_ACCESS=$GATEWAY_0_ACCESS

# workaround for issues with automatic accepting monitoring question
# with first run we need to accept question y/n about monitoring
mkdir -p ~/.config/storj/uplink/
touch ~/.config/storj/uplink/config.ini

uplinkng access save -f --name test-access --access $STORJ_ACCESS 

uplinkng mb "sj://$BUCKET/" --access $STORJ_ACCESS

uplinkng cp "$SRC_DIR/small-upload-testfile"        "sj://$BUCKET/" --progress=false --access $STORJ_ACCESS
uplinkng cp "$SRC_DIR/big-upload-testfile"          "sj://$BUCKET/" --progress=false --access $STORJ_ACCESS
uplinkng cp "$SRC_DIR/multisegment-upload-testfile" "sj://$BUCKET/" --progress=false --access $STORJ_ACCESS
uplinkng cp "$SRC_DIR/diff-size-segments"           "sj://$BUCKET/" --progress=false --access $STORJ_ACCESS

uplinkng access save -f --name named-access --access $STORJ_ACCESS
FILES=$(STORJ_ACCESS= uplinkng --access named-access ls "sj://$BUCKET" | tee $TMPDIR/list | wc -l)
EXPECTED_FILES="5"
if [ "$FILES" == $EXPECTED_FILES ]
then
    echo "listing returns $FILES files"
else
    echo "listing returns $FILES files but want $EXPECTED_FILES"
    exit 1
fi

SIZE_CHECK=$(cat "$TMPDIR/list" | awk '{if($4 == "0") print "invalid size";}')
if [ "$SIZE_CHECK" != "" ]
then
    echo "listing returns invalid size for one of the objects:"
    cat "$TMPDIR/list"
    exit 1
fi

uplinkng ls "sj://$BUCKET/non-existing-prefix" --access $STORJ_ACCESS

uplinkng cp  "sj://$BUCKET/small-upload-testfile"        "$DST_DIR" --progress=false  --access $STORJ_ACCESS
uplinkng cp  "sj://$BUCKET/big-upload-testfile"          "$DST_DIR" --progress=false  --access $STORJ_ACCESS
uplinkng cp  "sj://$BUCKET/multisegment-upload-testfile" "$DST_DIR" --progress=false  --access $STORJ_ACCESS
uplinkng cp  "sj://$BUCKET/diff-size-segments"           "$DST_DIR" --progress=false  --access $STORJ_ACCESS

uplinkng ls "sj://$BUCKET/small-upload-testfile" --access $STORJ_ACCESS | grep "small-upload-testfile"

uplinkng rm "sj://$BUCKET/small-upload-testfile"        --access $STORJ_ACCESS
uplinkng rm "sj://$BUCKET/big-upload-testfile"          --access $STORJ_ACCESS
uplinkng rm "sj://$BUCKET/multisegment-upload-testfile" --access $STORJ_ACCESS
uplinkng rm "sj://$BUCKET/diff-size-segments"           --access $STORJ_ACCESS

uplinkng ls "sj://$BUCKET" --access $STORJ_ACCESS
uplinkng ls -x true "sj://$BUCKET" --access $STORJ_ACCESS

uplinkng rb "sj://$BUCKET" --access $STORJ_ACCESS

compare_files "$SRC_DIR/small-upload-testfile"        "$DST_DIR/small-upload-testfile"
compare_files "$SRC_DIR/big-upload-testfile"          "$DST_DIR/big-upload-testfile"
compare_files "$SRC_DIR/multisegment-upload-testfile" "$DST_DIR/multisegment-upload-testfile"
compare_files "$SRC_DIR/diff-size-segments"           "$DST_DIR/diff-size-segments"

# test deleting non empty bucket with --force flag
uplinkng mb "sj://$BUCKET/" --access $STORJ_ACCESS

for i in $(seq -w 1 16); do
  uplinkng cp "$SRC_DIR/small-upload-testfile" "sj://$BUCKET/small-file-$i" --progress=false --access $STORJ_ACCESS
done

uplinkng rb "sj://$BUCKET" --force --access $STORJ_ACCESS

if [ "$(uplinkng ls --access $STORJ_ACCESS | wc -l)" != "0" ]; then
  echo "an integration test did not clean up after itself entirely"
  exit 1
fi
