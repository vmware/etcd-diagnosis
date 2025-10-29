#!/usr/bin/env bash

# Please run this script in the root directory of the repo.

set -euo pipefail

GIT_SHA=$(git rev-parse --short HEAD || echo "GitNotFound")
VERSION_SHA="github.com/vmware/etcd-diagnosis/version.GitSHA"

# use go env if noset
GOOS=${GOOS:-$(go env GOOS)}
GOARCH=${GOARCH:-$(go env GOARCH)}

GO_BUILD_FLAGS=${GO_BUILD_FLAGS:-}
GO_LDFLAGS=(${GO_LDFLAGS:-} "-X=${VERSION_SHA}=${GIT_SHA}")

export CGO_ENABLED=0
export GOOS
export GOARCH

rm -f ./bin/etcd-diagnosis

go build $GO_BUILD_FLAGS -trimpath -installsuffix=cgo "-ldflags=${GO_LDFLAGS[*]}" -o ./bin/etcd-diagnosis
