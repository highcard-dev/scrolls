
# Scrolls Registry

In this project you will find:
- The latest version of all the scrolls in the registry 
- The `scroll-registry` CLI tool that enables:
    - Packaging and pushing scrolls into the registry
    - Pushing metadata files to the registry (ie. translations)

## Usage/Examples

Copy the .env.example to .env and fill it with the credentials in Bitwarden under the name "scroll-registry-env"

Once you are done with it you can then run to package all the files under the `scrolls` folder and push into the registry: 
```bash
make push-all
```

## FAQ

#### How can I add a new game?

Within the scrolls folder copy and paste the `.sample` folder and rename it to the name of your game.
It's important that the folder structure within the `scrolls` folder is kept in the current standard for the CLI tool to work properly.

#### How can I update a scroll?

Perform the changes in the variant of the scroll you want to change. Once you are done, bump the version in the `scroll.yaml` file in the root folder of the variant.
Once you have done that, create a PR so that the scroll is made available for you to test in the staging environment.
Once you are done - merge it and it will be automatically published as a new version for all the current scroll deployments.

Answer 2


## Authors

- [@thiduzz](https://github.com/thiduzz)
- [@MarcStdt](https://github.com/MarcStdt)
- [@adrianmxb](https://github.com/adrianmxb)