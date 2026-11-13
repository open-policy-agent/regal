#!/bin/sh

# pre-commit helper: download the release binary matching the pinned hook
# revision if missing, then lint with it.

set -e

REPO=open-policy-agent/regal
BASE_URL=https://github.com/${REPO}

SCRIPT=$(readlink -f "$0")
SCRIPTPATH=$(dirname "$SCRIPT")

# Which Regal version to use. An explicit REGAL_VERSION always wins.
# Otherwise use the release tag of the checked-out hook revision, so pinning
# the hook pins Regal too. pre-commit checkouts carry no tags, so map the
# commit back to its tag with ls-remote. When the revision isn't a release
# tag, a previously downloaded binary is reused if there is one, and the
# latest release is downloaded only as a last resort.
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
            # A commit can carry several tags; pick one deterministically:
            # de-duplicate, prefer a release tag (v followed by a digit),
            # and fall back to "latest" below if none matches.
            TAG=$(git ls-remote --tags "${ORIGIN_URL}" 2>/dev/null | grep -F "${HEAD_SHA}" | sed -n 's|.*refs/tags/||p' | sed 's|\^{}$||' | sort -u | grep -E '^v[0-9]' | head -n 1) || TAG=""
        fi
    fi

    if [ -n "${TAG}" ]; then
        echo "${TAG}"
    else
        echo "latest"
    fi
}

# Version resolved for this checkout on an earlier run, or nothing. The memo
# pairs the version with the checkout's HEAD SHA, so a newly pinned revision
# re-resolves instead of trusting a stale version.
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

# An explicit REGAL_VERSION always wins, including over the cached-binary
# fallback below.
if [ -n "${REGAL_VERSION}" ]; then
    REGAL_VERSION_EXPLICIT=1
fi

if [ -z "${REGAL_VERSION_EXPLICIT:-}" ]; then
    # Warm runs reuse the memoized version instead of calling ls-remote again.
    REGAL_VERSION=$(memoized_version) || REGAL_VERSION=""
fi
if [ -z "${REGAL_VERSION:-}" ]; then
    REGAL_VERSION=$(resolve_version)
    if [ -z "${REGAL_VERSION_EXPLICIT:-}" ] && [ "${REGAL_VERSION}" != "latest" ]; then
        # Record the resolved version for warm runs. "latest" is never
        # memoized, so an undetermined version is re-resolved next time.
        HEAD_SHA=$(git -C "${SCRIPTPATH}" rev-parse HEAD 2>/dev/null) || HEAD_SHA=""
        if [ -n "${HEAD_SHA}" ]; then
            printf '%s\n%s\n' "${HEAD_SHA}" "${REGAL_VERSION}" > "${SCRIPTPATH}/.regal-version" 2>/dev/null || true
        fi
    fi
fi

# One cached binary per version, so a version change downloads its own
# binary. (The fallback below is the exception: when the version can't be
# determined, the newest cached versioned binary is reused instead.)
BIN_PATH="${SCRIPTPATH}/regal-${REGAL_VERSION}"

# With no explicit version and an undetermined ("latest") version, prefer a
# previously downloaded versioned binary: a "latest" download fails outright
# offline (curl under set -e aborts the commit) even though a usable binary
# is already cached.
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

# Download only when the cached binary is missing or reports the wrong
# version (e.g. an interrupted earlier download left a bad file). A
# "latest" binary can't be checked against a moving target, so its cache
# is trusted as-is.
needs_download()
{
    if [ ! -x "${BIN_PATH}" ]; then
        return 0
    fi
    if [ "${REGAL_VERSION}" = "latest" ]; then
        return 1
    fi
    INSTALLED_VERSION=$(REGAL_DISABLE_VERSION_CHECK=1 "${BIN_PATH}" version 2>/dev/null | sed -n 's/^Version:[[:space:]]*//p')
    # Binaries report the version without the "v" prefix ("0.25.0" for tag
    # "v0.25.0"); strip it before comparing, or every run re-downloads.
    [ "${INSTALLED_VERSION}" != "${REGAL_VERSION#v}" ]
}

if needs_download; then download; fi
"${BIN_PATH}" $@
