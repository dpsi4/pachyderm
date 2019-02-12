#!/bin/bash
# Build the 'split' pipeline and supervisor such that they can be run statically
# (i.e. in a scratch containiner, with no libc), so that their docker images are
# as small as possible.
#
# **Compiled binaries will be stored in ./_out**

set -ex

# Clear _out, to hold build output
rm -rf ./_out || true
mkdir _out

# Setup build command. The linker flags, along with CGO_ENABLED=0 (set below)
# tell the go compiler to build a fully static binary (see comment at top)
LD_FLAGS="-extldflags -static"
BUILD_PATH=github.com/pachyderm/pachyderm/src/testing/loadtest/split
BUILD_CMD="
go install -a -ldflags \"${LD_FLAGS}\" ./${BUILD_PATH}/cmd/pipeline && \
go install -a -ldflags \"${LD_FLAGS}\" ./${BUILD_PATH}/cmd/supervisor && \
mv ../bin/* /out/"

# Run compilation inside golang container, with pachyderm mounted
# Explanation of mounts
# ---------------------
# - _out is where the binaries built in this docker image are written out for
#   us to use:
#     /out <- ./_out
#
# - $HOME/.cache/go-build holds cached docker builds; mounting it makes builds
#   faster:
#     /root/.cache/go-build <- $HOME/.cache/go-build
#
# - Mouting the whole pachyderm repo (including the pachyderm client and this
#   benchmark) makes this benchmark available to be built. Mounting all of it in
#   together (rather than e.g. mounting just this dir and having the pachdyerm
#   repo in ./vendor) means that parallel changes can be made to the benchmark,
#   server, and client simultaneously:
#     /go/src/.../pachyderm <- $GOPATH/src/.../pachyerm/
#
# - Mounting src/server/vendor into src/client/vendor is necessary to build the
#   pachyderm client (which the 'split' supervisor depends on, and which has
#   dependencies but has no vendor directory). It accomplishes a similar goal to
#   the mainline pachyderm's 'make docker-build' directive, which symlinks
#   src/server/vendor into the client:
#     /go/src/../pachyderm/src/client/vendor <- $GOPATH/.../pachyderm/src/server/vendor
#
# - Mounting ./_out into src/client/vendor/...pachyderm avoids a stupid bug where
#   src/client/vendor/github.com/pachyderm/pachyderm/src/client can fail to
#   build because it (a vendored dependency) doesn't have the right
#   dependencies to build. We prevent the go compiler from trying to build the
#   client inside the vendored dependencies of the client by mounting a
#   directory over it that contains no go code:
#     /go/src/../pachyderm/src/client/vendor/github.com/pachyderm/ <- ./_out
docker run \
  -w /go/src \
  -e CGO_ENABLED=0 \
  -v "${PWD}/_out:/out" \
  -v "${HOME}/.cache/go-build:/root/.cache/go-build" \
  -v "${GOPATH}/src/github.com/pachyderm/pachyderm:/go/src/github.com/pachyderm/pachyderm:ro" \
  -v "${GOPATH}/src/github.com/pachyderm/pachyderm/src/server/vendor:/go/src/github.com/pachyderm/pachyderm/src/client/vendor:ro" \
  -v "${PWD}/_out:/go/src/github.com/pachyderm/pachyderm/src/client/vendor/github.com/pachyderm:ro" \
  golang:1.11 /bin/sh -c "${BUILD_CMD}"
