#!/bin/sh

set -ex 

if [ -z ${VERSION+x} ]
then
    echo 'Error! $VERSION is required.'
    exit 64
fi

echo $VERSION

goreleaser check

tag_and_push() {
    local component=$1
    git tag -a $component/$VERSION -m "release $component/$VERSION"
    git push origin $component/$VERSION
    # echo "release $component/$VERSION"
}


tag_and_push "grid-client"
tag_and_push "grid-proxy"
tag_and_push "rmb-sdk-go"
tag_and_push "activation-service"

# # main
git tag -a $VERSION -m "release $VERSION"
git push origin $VERSION
