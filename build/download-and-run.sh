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
            # A commit can carry more than one tag (a release tag plus a
            # nightly/latest alias, or an annotated tag plus its lightweight
            # alias). Pick deterministically: de-duplicate, then prefer a
            # release tag (v followed by a digit). If none matches, fall back
            # to "latest" below instead of resolving to a non-release ref
            # whose download URL would 404.
            TAG=$(git ls-remote --tags "${ORIGIN_URL}" 2>/dev/null | grep -F "${HEAD_SHA}" | sed -n 's|.*refs/tags/||p' | sed 's|\^{}$||' | sort -u | grep -E '^v[0-9]' | head -n 1) || TAG=""
        fi
    fi

    if [ -n "${TAG}" ]; then
        echo "${TAG}"
    else
        echo "latest"
    fi
}

# Remember whether the version was explicitly requested: an explicit
# REGAL_VERSION always wins, including over the cached-binary fallback below.
if [ -n "${REGAL_VERSION}" ]; then
    REGAL_VERSION_EXPLICIT=1
fi

REGAL_VERSION=$(resolve_version)

# Keep one cached binary per version so a previously downloaded binary for a
# different version is never reused after the pinned version changes.
BIN_PATH="${SCRIPTPATH}/regal-${REGAL_VERSION}"

# When no explicit version was requested and the revision isn't a release
# tag, the version above fell back to "latest". Prefer a previously
# downloaded versioned binary in that case instead of downloading "latest":
# the old script reused an existing binary, and a "latest" download fails
# outright when the network is unavailable (curl under set -e aborts the
# commit) even though a usable binary is already cached. An explicitly set
# REGAL_VERSION still wins over the cache.
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

# Only download when there is no cached binary for the pinned version, or the
# cached binary doesn't report the pinned version (e.g. an interrupted earlier
# download left a bad file behind). A "latest" binary can't be cheaply verified
# against a moving target, so the cache is trusted for it.
needs_download()
{
    if [ ! -x "${BIN_PATH}" ]; then
        return 0
    fi
    if [ "${REGAL_VERSION}" = "latest" ]; then
        return 1
    fi
    INSTALLED_VERSION=$(REGAL_DISABLE_VERSION_CHECK=1 "${BIN_PATH}" version 2>/dev/null | sed -n 's/^Version:[[:space:]]*//p')
    # Release binaries report the version without the "v" prefix (e.g. "0.25.0"
    # for tag "v0.25.0"), so strip it before comparing, or every run would
    # re-download instead of using the cached binary.
    [ "${INSTALLED_VERSION}" != "${REGAL_VERSION#v}" ]
}

if needs_download; then download; fi
"${BIN_PATH}" $@
