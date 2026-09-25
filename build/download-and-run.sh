#!/bin/sh

# pre-commit helper to download the release binary matching the pinned hook
# revision if missing before executing linting with it.

set -e

REPO=open-policy-agent/regal
BASE_URL=https://github.com/${REPO}

SCRIPT=$(readlink -f "$0")
SCRIPTPATH=$(dirname "$SCRIPT")

# Resolve which Regal version to download. An explicitly set REGAL_VERSION always
# wins. Otherwise, when run as a pre-commit hook, this script lives in the hook
# repository checked out at the revision pinned in .pre-commit-config.yaml, so
# default to the release tag of that revision — pinning the hook then pins the
# Regal version too. pre-commit's checkout doesn't keep tags around, so map the
# checked-out commit back to its tag via ls-remote. Fall back to the latest
# release when the revision isn't a release tag (e.g. a branch or commit SHA)
# or when the version can't be determined.
resolve_version()
{
    if [ -n "${REGAL_VERSION}" ]; then
        echo "${REGAL_VERSION}"
        return
    fi

    TAG=$(git -C "${SCRIPTPATH}" describe --tags --exact-match 2>/dev/null) || TAG=""
    if [ -z "${TAG}" ]; then
        HEAD_SHA=$(git -C "${SCRIPTPATH}" rev-parse HEAD 2>/dev/null) || HEAD_SHA=""
        ORIGIN_URL=$(git -C "${SCRIPTPATH}" config --get remote.origin.url 2>/dev/null) || ORIGIN_URL=""
        if [ -n "${HEAD_SHA}" ] && [ -n "${ORIGIN_URL}" ]; then
            TAG=$(git ls-remote --tags "${ORIGIN_URL}" 2>/dev/null | grep -F "${HEAD_SHA}" | sed -n 's|.*refs/tags/||p' | sed 's|\^{}$||' | head -n 1) || TAG=""
        fi
    fi

    if [ -n "${TAG}" ]; then
        echo "${TAG}"
    else
        echo "latest"
    fi
}

REGAL_VERSION=$(resolve_version)

# Keep one cached binary per version so a previously downloaded binary for a
# different version is never reused after the pinned version changes.
BIN_PATH="${SCRIPTPATH}/regal-${REGAL_VERSION}"

download()
{
    DETECTED_SYSTEM=$(uname -s)
    DETECTED_ARCHITECTURE=$(uname -m)

    SYSTEM=${REGAL_SYSTEM:-${DETECTED_SYSTEM}}
    ARCHITECTURE=${REGAL_ARCHITECTURE:-${DETECTED_ARCHITECTURE}}

    echo "Downloading regal for ${SYSTEM} ${ARCHITECTURE}, ${REGAL_VERSION}…"
    BINARY_URL=${BASE_URL}/releases/${REGAL_VERSION}/download/regal_${SYSTEM}_${ARCHITECTURE}
    curl --fail -Lo "${BIN_PATH}" ${BINARY_URL}
    chmod +x "${BIN_PATH}"
}

if [ ! -x "${BIN_PATH}" ]; then download; fi
"${BIN_PATH}" $@
