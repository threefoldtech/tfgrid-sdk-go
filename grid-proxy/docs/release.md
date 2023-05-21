## Release Grid-Proxy
To release a new version of the Grid-Proxy component, follow these steps:

Update the `appVersion` field in the `charts/Chart.yaml` file. This field should reflect the new version number of the release.

The release process includes generating and pushing a Docker image with the latest GitHub tag. This step is automated through the `gridproxy-release.yml` workflow.

Trigger the `gridproxy-release.yml` workflow by pushing the desired tag to the repository. This will initiate the workflow, which will generate the Docker image based on the tag and push it to the appropriate registry.

## Debugging
In the event that the workflow does not run automatically after pushing the tag and making the release, you can manually execute it using the GitHub Actions interface. Follow these steps:

Go to the [GitHub Actions page](https://github.com/threefoldtech/tfgrid-sdk-go/actions/workflows/gridproxy-release.yml) for the Grid-Proxy repository.

Locate the workflow named gridproxy-release.yml.

Trigger the workflow manually by selecting the "Run workflow" option.