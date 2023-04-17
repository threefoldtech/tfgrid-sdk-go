
# Release

let's say the next tag is `v1.0.0`, release will be:

## Grid-client

- Create a tag `git tag -a grid-client/v1.0.0 -m "release grid-client/v1.0.0"`
- Push the tag `git push origin grid-client/v1.0.0`

## Grid-proxy

- Create a tag `git tag -a grid-proxy/v1.0.0 -m "release grid-proxy/v1.0.0"`
- Push the tag `git push origin grid-proxy/v1.0.0`

## RMB-sdk-go

- Create a tag `git tag -a rmb-sdk-go/v1.0.0 -m "release rmb-sdk-go/v1.0.0"`
- Push the tag `git push origin rmb-sdk-go/v1.0.0`

## Main release

- Check `goreleaser check`
- Create a tag `git tag -a v1.0.0 -m "release v1.0.0"`
- Push the tag `git push origin v1.0.0`
- the release workflow will release the tag automatically
