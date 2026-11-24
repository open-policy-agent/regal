#!/bin/sh

# Download the release binary for the pinned hook revision, then lint with it.

set -e

REPO=open-policy-agent/regal
BASE_URL=https://github.com/${REPO}

SCRIPT=$(readlink -f "$0")
SCRIPTPATH=$(dirname "$SCRIPT")

# Which Regal version to use: an explicit REGAL_VERSION wins, otherwise the
# release tag of the checked-out hook revision. pre-commit checkouts carry no
# tags, so the commit is mapped back to its tag with ls-remote, or — when
# ls-remote is unreachable — with a tag recorded by an earlier fetch. If the
# revision is not a release tag, the version is "latest".
#
# A tag recorded by an earlier fetch (FETCH_HEAD) can still name the pinned
# revision when ls-remote is unreachable: the fetch that brought the tag in
# records its name, and the tag object dereferences to the checked-out
# commit. Only a tag that resolves to HEAD is trusted, so a stale fetch
# record never overrides the actual checkout. Release tags are preferred.
fetched_tag()
{
    HEAD_SHA=$(git -C "${SCRIPTPATH}" rev-parse HEAD 2>/dev/null) || HEAD_SHA=""
    [ -n "${HEAD_SHA}" ] || return 1
    FETCH_HEAD_FILE=$(git -C "${SCRIPTPATH}" rev-parse --git-path FETCH_HEAD 2>/dev/null) || FETCH_HEAD_FILE=""
    case "${FETCH_HEAD_FILE}" in
        /*) ;;
        *) FETCH_HEAD_FILE="${SCRIPTPATH}/${FETCH_HEAD_FILE}" ;;
    esac
    [ -f "${FETCH_HEAD_FILE}" ] || return 1
    FALLBACK=""
    while IFS= read -r line || [ -n "${line}" ]; do
        case "${line}" in
            *"	tag '"*)
                SHA=$(printf '%s\n' "${line}" | sed -n 's/^[[:space:]]*\([0-9a-f][0-9a-f]*\).*/\1/p')
                NAME=$(printf '%s\n' "${line}" | sed -n "s/.*tag '\([^']*\)'.*/\1/p")
                [ -n "${SHA}" ] && [ -n "${NAME}" ] || continue
                COMMIT=$(git -C "${SCRIPTPATH}" rev-parse "${SHA}^{commit}" 2>/dev/null) || continue
                [ "${COMMIT}" = "${HEAD_SHA}" ] || continue
                case "${NAME}" in
                    v[0-9]*)
                        echo "${NAME}"
                        return 0
                        ;;
                    *)
                        [ -z "${FALLBACK}" ] && FALLBACK="${NAME}"
                        ;;
                esac
                ;;
        esac
    done < "${FETCH_HEAD_FILE}"
    [ -n "${FALLBACK}" ] && echo "${FALLBACK}"
}

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
        # ls-remote unreachable (offline, for example): fall back to a tag
        # recorded by an earlier fetch before giving up on the pin.
        TAG=$(fetched_tag) || TAG=""
    fi

    if [ -n "${TAG}" ]; then
        echo "${TAG}"
    else
        echo "latest"
    fi
}

# Version resolved for this checkout on an earlier run. The memo pairs it
# with the HEAD SHA, so a newly pinned revision re-resolves.
memoized_version()
{
    MEMO="${SCRIPTPATH}/.regal-version"
    [ -f "${MEMO}" ] || return 1
    MEMO_SHA=$(sed -n '1p' "${MEMO}" 2>/dev/null)
    MEMO_VERSION=$(sed -n '2p' "${MEMO}" 2>/dev/null)
    HEAD_SHA=$(git -C "${SCRIPTPATH}" rev-parse HEAD 2>/dev/null) || HEAD_SHA=""
    [ -n "${MEMO_SHA}" ] && [ -n "${MEMO_VERSION}" ] && [ "${MEMO_VERSION}" != "latest" ] && [ "${MEMO_SHA}" = "${HEAD_SHA}" ] || return 1
    echo "${MEMO_VERSION}"
}

# An explicit REGAL_VERSION always wins, including over the cached fallback.
if [ -n "${REGAL_VERSION}" ]; then
    REGAL_VERSION_EXPLICIT=1
fi

if [ -z "${REGAL_VERSION_EXPLICIT:-}" ]; then
    # Warm runs reuse the memoized version instead of calling ls-remote.
    REGAL_VERSION=$(memoized_version) || REGAL_VERSION=""
fi
if [ -z "${REGAL_VERSION:-}" ]; then
    REGAL_VERSION=$(resolve_version)
    if [ -z "${REGAL_VERSION_EXPLICIT:-}" ] && [ "${REGAL_VERSION}" != "latest" ]; then
        # "latest" is never memoized, so an undetermined version re-resolves.
        HEAD_SHA=$(git -C "${SCRIPTPATH}" rev-parse HEAD 2>/dev/null) || HEAD_SHA=""
        if [ -n "${HEAD_SHA}" ]; then
            printf '%s\n%s\n' "${HEAD_SHA}" "${REGAL_VERSION}" > "${SCRIPTPATH}/.regal-version" 2>/dev/null || true
        fi
    fi
fi

# One cached binary per version. The fallback below is the only exception.
BIN_PATH="${SCRIPTPATH}/regal-${REGAL_VERSION}"

# No version could be determined ("latest") and none was set explicitly:
# reuse the newest cached versioned binary instead of downloading, which
# would fail outright offline. A resolved pin never reaches this fallback.
if [ -z "${REGAL_VERSION_EXPLICIT:-}" ] && [ "${REGAL_VERSION}" = "latest" ]; then
    CACHED=""
    for candidate in "${SCRIPTPATH}"/regal-v*; do
        [ -x "${candidate}" ] || continue
        if [ -z "${CACHED}" ] || [ "${candidate}" -nt "${CACHED}" ]; then
            CACHED="${candidate}"
        fi
    done
    if [ -n "${CACHED}" ]; then
        echo "regal version could not be determined; using cached ${CACHED}" >&2
        BIN_PATH="${CACHED}"
    fi
fi

download()
{
    DETECTED_SYSTEM=$(uname -s)
    DETECTED_ARCHITECTURE=$(uname -m)

    SYSTEM=${REGAL_SYSTEM:-${DETECTED_SYSTEM}}
    ARCHITECTURE=${REGAL_ARCHITECTURE:-${DETECTED_ARCHITECTURE}}

    echo "Downloading regal for ${SYSTEM} ${ARCHITECTURE}, ${REGAL_VERSION}…"
    if [ "${REGAL_VERSION}" = "latest" ]; then
        # GitHub only serves the "latest" pseudo-release through this endpoint.
        BINARY_URL=${BASE_URL}/releases/latest/download/regal_${SYSTEM}_${ARCHITECTURE}
    else
        BINARY_URL=${BASE_URL}/releases/download/${REGAL_VERSION}/regal_${SYSTEM}_${ARCHITECTURE}
    fi
    curl --fail -Lo "${BIN_PATH}" ${BINARY_URL}
    chmod +x "${BIN_PATH}"
}

# Download when the cached binary is missing or reports the wrong version
# (e.g. an interrupted earlier download). A "latest" binary can't be checked
# against a moving target, so it is trusted as-is.
needs_download()
{
    if [ ! -x "${BIN_PATH}" ]; then
        return 0
    fi
    if [ "${REGAL_VERSION}" = "latest" ]; then
        return 1
    fi
    INSTALLED_VERSION=$(REGAL_DISABLE_VERSION_CHECK=1 "${BIN_PATH}" version 2>/dev/null | sed -n 's/^Version:[[:space:]]*//p')
    # Binaries report the version without the "v" prefix; strip it first.
    [ "${INSTALLED_VERSION}" != "${REGAL_VERSION#v}" ]
}

if needs_download; then download; fi
"${BIN_PATH}" $@
