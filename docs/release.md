
# Release

- Export `$VERSION` env variable to the version you want
- Run `make release`

## Release without script

let's say the next tag is `v1.0.0`, release will be:

### Grid-client

- Create a tag `git tag -a grid-client/v1.0.0 -m "release grid-client/v1.0.0"`
- Push the tag `git push origin grid-client/v1.0.0`

### Grid-proxy

- Create a tag `git tag -a grid-proxy/v1.0.0 -m "release grid-proxy/v1.0.0"`
- Push the tag `git push origin grid-proxy/v1.0.0`
For Further info check Grid-proxy release [docs](../grid-proxy/docs/release.md).

### RMB-sdk-go

- Create a tag `git tag -a rmb-sdk-go/v1.0.0 -m "release rmb-sdk-go/v1.0.0"`
- Push the tag `git push origin rmb-sdk-go/v1.0.0`

### Main release

- Check `goreleaser check`
- Create a tag `git tag -a v1.0.0 -m "release v1.0.0"`
- Push the tag `git push origin v1.0.0`
- the release workflow will release the tag automatically

## Tags Convention
The following convention should be followed for tagging in this project:

Release Tags: For release names and GitHub tags, the tag format should be prefixed with v0.0.0. For example, a release tag could be v1.2.3, where 1.2.3 represents the version number of the release.

Docker Image Tags: For generated Docker images, such as in the tfgridproxy component, the tag format should only include the tag number without the v prefix. For example, a Docker image tag could be 0.0.0, representing the specific version of the image.

Following this convention will help maintain consistency and clarity in tagging across all the grid components.