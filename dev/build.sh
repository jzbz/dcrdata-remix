#!/bin/bash
# Usage: build.sh [dcrdata_root] [destination_folder]
#
#   build.sh performs the following actions:
#       1. Compile go code, generating the main binary.
#       2. Gzip the compressible static assets.
#       3. (Optional) Install everything.
#
#   The front end is plain CSS + native ES modules served straight from
#   public/, so there is no bundler step.
#
#   When run with no arguments, build.sh will use the repository root as the
#   root folder. If not running from a git repository, the dcrdata_root folder
#   must be specified.
#
#   Specify destination_folder to install the dcrdata executable and the static
#   assets (public and views_v2 folders). When destination_folder is omitted,
#   the generated files are not installed.
#
#   Note that this script uses 7za to Gzip static assets. The standard gzip
#   utility is not used since 7za compression rates are slightly better even for
#   the gz format.
#
# Copyright (c) 2018-2020, The Decred developers.
# See LICENSE for details.

REPO=`git rev-parse --show-toplevel 2> /dev/null`
if [[ $? != 0 ]]; then
    REPO=
fi

ROOT=${1:-$REPO}

if [[ -z "$ROOT" ]]; then
    echo "Not in git repository. You must specify the dcrdata root folder as the first argument!"
    exit 1
fi

set -e

# Delete the old dcrdata binary that is now under cmd/dcrdata.
rm -f ${ROOT}/dcrdata

pushd $ROOT/cmd/dcrdata > /dev/null

echo "Building the dcrdata binary..."
go build -v

echo "Gzipping assets for use with gzip_static..."
find ./public -type f -name "*.gz" -execdir rm {} \;
# The find arguments live in an array: a quoted string expansion would keep
# literal quote characters in the -name patterns, so the exclusions would
# never match and every binary asset would be recompressed each build.
FINDARGS=(./public -type f -not -name '*.gz' -not -name '*.scss' -not -name '*.png' -not -name '*.woff2')
# Use GNU parallel if it is installed.
if [ -x "$(command -v parallel)" ]; then
    if [ -x "$(command -v 7za)" ]; then
        find "${FINDARGS[@]}" | parallel --will-cite --bar 7za a -tgzip -mx=9 -mpass=13 {}.gz {} > /dev/null
    else
        find "${FINDARGS[@]}" | parallel --will-cite --bar gzip -k9f {} > /dev/null
    fi
elif [ -x "$(command -v 7za)" ]; then
    find "${FINDARGS[@]}" -execdir 7za a -tgzip -mx=9 -mpass=13 {}.gz {} \; > /dev/null
else
    find "${FINDARGS[@]}" -execdir gzip -k9f {} \; > /dev/null
fi

DEST=$2

if [[ -n "$DEST" ]]; then
    # deploy.sh provisions user and group "dcrdata"; no "decred" group exists.
    sudo install -m 754 -o dcrdata -g dcrdata ./dcrdata "${DEST}/"
    sudo rm -rf "${DEST}/views_v2" "${DEST}/public"
    sudo cp -R views_v2 public "${DEST}/"
fi

popd > /dev/null

echo "Done"
