#!/bin/sh

# pre-commit helper: download the release binary if missing, then lint with it.

set -e

REPO=open-policy-agent/regal
BASE_URL=https://github.com/${REPO}

SCRIPT=$(readlink -f "$0")
SCRIPTPATH=$(dirname "$SCRIPT")
BIN_PATH="${SCRIPTPATH}/regal"

# A tag recorded by an earlier fetch can still name the pinned revision when
# ls-remote is unreachable. Only a tag resolving to the checked-out commit
# is trusted, so a stale fetch record never overrides the checkout.
fetched_tag()
{
    HEAD_SHA=$(git -C "${SCRIPTPATH}" rev-parse HEAD 2>/dev/null) || return 1
    FETCH_HEAD_FILE=$(git -C "${SCRIPTPATH}" rev-parse --git-path FETCH_HEAD 2>/dev/null) || return 1
    case "${FETCH_HEAD_FILE}" in
        /*) ;;
        *) FETCH_HEAD_FILE="${SCRIPTPATH}/${FETCH_HEAD_FILE}" ;;
    esac
    [ -f "${FETCH_HEAD_FILE}" ] || return 1
    while IFS= read -r line || [ -n "${line}" ]; do
        case "${line}" in
            *"tag '"*)
                SHA=$(printf '%s\n' "${line}" | sed -n 's/^[[:space:]]*\([0-9a-f][0-9a-f]*\).*/\1/p')
                NAME=$(printf '%s\n' "${line}" | sed -n "s/.*tag '\([^']*\)'.*/\1/p")
                [ -n "${SHA}" ] && [ -n "${NAME}" ] || continue
                case "${NAME}" in
                    v[0-9]*) ;;
                    *) continue ;;
                esac
                COMMIT=$(git -C "${SCRIPTPATH}" rev-parse "${SHA}^{commit}" 2>/dev/null) || continue
                if [ "${COMMIT}" = "${HEAD_SHA}" ]; then
                    echo "${NAME}"
                    return 0
                fi
                ;;
        esac
    done < "${FETCH_HEAD_FILE}"
    return 1
}

# Which version to use: an explicit REGAL_VERSION wins, otherwise the
# release tag of the checked-out revision. Pre-commit checkouts carry no
# tags, so the commit is mapped back to its tag with ls-remote, or — when
# ls-remote is unreachable — with a tag recorded by an earlier fetch.
# Anything else wants no particular version: the cached binary is used.
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
            # Several tags can point at one commit; prefer a release tag.
            TAG=$(git ls-remote --tags "${ORIGIN_URL}" 2>/dev/null | grep -F "${HEAD_SHA}" | sed -n 's|.*refs/tags/||p' | sed 's|\^{}$||' | sort -u | grep -E '^v[0-9]' | head -n 1) || TAG=""
        fi
    fi
    if [ -z "${TAG}" ]; then
        TAG=$(fetched_tag) || TAG=""
    fi
    echo "${TAG}"
}

VERSION=$(resolve_version)

download()
{
    DETECTED_SYSTEM=$(uname -s)
    DETECTED_ARCHITECTURE=$(uname -m)

    SYSTEM=${REGAL_SYSTEM:-${DETECTED_SYSTEM}}
    ARCHITECTURE=${REGAL_ARCHITECTURE:-${DETECTED_ARCHITECTURE}}

    echo "Downloading regal for ${SYSTEM} ${ARCHITECTURE}, ${VERSION:-latest}…"
    if [ -n "${VERSION}" ] && [ "${VERSION}" != "latest" ]; then
        BINARY_URL=${BASE_URL}/releases/download/${VERSION}/regal_${SYSTEM}_${ARCHITECTURE}
    else
        # GitHub only serves the "latest" pseudo-release through this endpoint.
        BINARY_URL=${BASE_URL}/releases/latest/download/regal_${SYSTEM}_${ARCHITECTURE}
    fi
    curl --fail -Lo "${BIN_PATH}" ${BINARY_URL}
    chmod +x "${BIN_PATH}"
}

# Download if there is no binary, or if a version is wanted and the cached
# binary reports a different one. With no version wanted, the cached binary
# is used as-is, so the hook keeps working offline.
needs_download()
{
    [ ! -x "${BIN_PATH}" ] && return 0
    [ -z "${VERSION}" ] && return 1
    INSTALLED=$(REGAL_DISABLE_VERSION_CHECK=1 "${BIN_PATH}" version 2>/dev/null | sed -n 's/^Version:[[:space:]]*//p')
    [ "${INSTALLED}" != "${VERSION#v}" ]
}

if needs_download; then download; fi
"${BIN_PATH}" $@
