#!/bin/sh

if [ -z ${VERSION+x} ]
then
    echo 'Error! $VERSION is required.'
    exit 64
fi

echo $VERSION

goreleaser check

# grid client
git tag -a grid-client/$VERSION -m "release grid-client/$VERSION"
git push origin grid-client/$VERSION

# grid proxy
git tag -a grid-proxy/$VERSION -m "release grid-proxy/$VERSION"
git push origin grid-proxy/$VERSION

# rmb sdk go
git tag -a rmb-sdk-go/$VERSION -m "release rmb-sdk-go/$VERSION"
git push origin rmb-sdk-go/$VERSION

# main
git tag -a $VERSION -m "release $VERSION"
git push origin $VERSION
