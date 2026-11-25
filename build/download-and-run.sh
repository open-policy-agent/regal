#!/bin/sh

# pre-commit helper to run download latest release binary if missing before executing
# linting with it.

set -e

REPO=open-policy-agent/regal
BASE_URL=https://github.com/${REPO}

SCRIPT=$(readlink -f "$0")
SCRIPTPATH=$(dirname "$SCRIPT")
BIN_PATH="${SCRIPTPATH}/regal"

download()
{
    DETECTED_SYSTEM=$(uname -s)
    DETECTED_ARCHITECTURE=$(uname -m)

    SYSTEM=${REGAL_SYSTEM:-${DETECTED_SYSTEM}}
    ARCHITECTURE=${REGAL_ARCHITECTURE:-${DETECTED_ARCHITECTURE}}

    echo "Downloading regal for ${SYSTEM} ${ARCHITECTURE}, ${REGAL_VERSION:-latest}…"
    if [ -n "${REGAL_VERSION}" ] && [ "${REGAL_VERSION}" != "latest" ]; then
        BINARY_URL=${BASE_URL}/releases/download/${REGAL_VERSION}/regal_${SYSTEM}_${ARCHITECTURE}
    else
        BINARY_URL=${BASE_URL}/releases/latest/download/regal_${SYSTEM}_${ARCHITECTURE}
    fi
    curl --fail -Lo "${BIN_PATH}" ${BINARY_URL}
    chmod +x "${BIN_PATH}"
}

# Download if there is no binary, or if REGAL_VERSION is set and differs from
# the cached one. Releases report their version without the "v" prefix.
needs_download()
{
    [ ! -x "${BIN_PATH}" ] && return 0
    [ -z "${REGAL_VERSION}" ] && return 1
    [ "${REGAL_VERSION}" = "latest" ] && return 1
    INSTALLED=$(REGAL_DISABLE_VERSION_CHECK=1 "${BIN_PATH}" version 2>/dev/null | sed -n 's/^Version:[[:space:]]*//p')
    [ "${INSTALLED}" != "${REGAL_VERSION#v}" ]
}

if needs_download; then download; fi
"${BIN_PATH}" $@
