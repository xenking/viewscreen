#!/bin/sh

# Require environment variables.
if [ -z "${VIEWSCREEN_HTTP_HOST-}" ] ; then
    echo "Environment variable VIEWSCREEN_HTTP_HOST required. Exiting."
    exit 1
fi
# Optional environment variables.
if [ -z "${VIEWSCREEN_BACKLINK-}" ] ; then
    export VIEWSCREEN_BACKLINK=""
fi

if [ -z "${VIEWSCREEN_LETSENCRYPT-}" ] ; then
    export VIEWSCREEN_LETSENCRYPT="true"
fi

if [ -z "${VIEWSCREEN_HTTP_ADDR-}" ] ; then
    export VIEWSCREEN_HTTP_ADDR=":80"
fi

if [ -z "${VIEWSCREEN_TORRENT_ADDR-}" ] ; then
    export VIEWSCREEN_TORRENT_ADDR=":61337"
fi

if [ -z "${VIEWSCREEN_HTTP_PREFIX-}" ] ; then
    export VIEWSCREEN_HTTP_PREFIX="/app"
fi

if [ -z "${VIEWSCREEN_METADATA-}" ] ; then
    export VIEWSCREEN_METADATA="false"
fi

if [ -z "${DEBUG-}" ] ; then
    export DEBUG="false"
fi

if [ -z "${VIEWSCREEN_HTTP_USER-}" ] ; then
    export VIEWSCREEN_HTTP_USER="viewscreen"
fi

exec /usr/bin/viewscreen \
    "--http-host=${VIEWSCREEN_HTTP_HOST}" \
    "--http-addr=${VIEWSCREEN_HTTP_ADDR}" \
    "--http-prefix=${VIEWSCREEN_HTTP_PREFIX}" \
    "--http-username=${VIEWSCREEN_HTTP_USER}" \
    "--backlink=${VIEWSCREEN_BACKLINK}" \
    "--letsencrypt=${VIEWSCREEN_LETSENCRYPT}" \
    "--torrent-addr=${VIEWSCREEN_TORRENT_ADDR}" \
    "--metadata=${VIEWSCREEN_METADATA}" \
    "--http-username=${VIEWSCREEN_HTTP_USER}" \
    "--debug=${DEBUG}"
