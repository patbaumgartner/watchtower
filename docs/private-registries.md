Watchtower supports private Docker image registries. In many cases, accessing a private registry
requires a valid username and password (i.e., _credentials_). In order to operate in such an
environment, watchtower needs to know the credentials to access the registry. 

The credentials can be provided to watchtower in a configuration file called `config.json`.
There are two ways to generate this configuration file:

*   The configuration file can be created manually.
*   Call `docker login <REGISTRY_NAME>` and share the resulting configuration file.

### Create the configuration file manually
Create a new configuration file with the following syntax and a base64 encoded username and
password `auth` string:

```json
{
    "auths": {
        "<REGISTRY_NAME>": {
            "auth": "XXXXXXX"
        }
    }
}
```

`<REGISTRY_NAME>` needs to be replaced by the name of your private registry
(e.g., `my-private-registry.example.org`).

!!! info "Using private images on Docker Hub"
    To access private repositories on Docker Hub,
    `<REGISTRY_NAME>` should be `https://index.docker.io/v1/`.
    In this special case, the registry domain does not have to be specified
    in `docker run` or `docker-compose`. Like Docker, Watchtower will use the
    Docker Hub registry and its credentials when no registry domain is specified.
    
    <sub>Watchtower will recognize credentials with `<REGISTRY_NAME>` `index.docker.io`,
    but the Docker CLI will not.</sub>

!!! important "Using a private registry on a local host"
    To use a private registry hosted locally, make sure to correctly specify the registry host
    in both `config.json` and the `docker run` command or `docker-compose` file.
    Valid hosts are `localhost[:PORT]`, `HOST:PORT`,
    or any multi-part `domain.name` or IP-address with or without a port.
    
    Examples:
    * `localhost` -> `localhost/myimage`
    * `127.0.0.1` -> `127.0.0.1/myimage:mytag`
    * `host.domain` -> `host.domain/myorganization/myimage`
    * `other-lan-host:80` -> `other-lan-host:80/imagename:latest`

The required `auth` string can be generated as follows:

```bash
echo -n 'username:password' | base64
```

!!! info "Username and Password for GCloud"
    For gcloud, we'll use `_json_key` as our username and the content of `gcloudauth.json` as the password.
    ```
    bash echo -n "_json_key:$(cat gcloudauth.json)" | base64 -w0
    ```

When the watchtower Docker container is started, the created configuration file
(`<PATH>/config.json` in this example) needs to be passed to the container:

```bash
docker run [...] -v <PATH>/config.json:/config.json patbaumgartner/watchtower:latest
```

### Share the Docker configuration file

To pull an image from a private registry, `docker login` needs to be called first, to get access
to the registry. The provided credentials are stored in a configuration file called `<PATH_TO_HOME_DIR>/.docker/config.json`.
This configuration file can be directly used by watchtower. In this case, the creation of an
additional configuration file is not necessary.

When the Docker container is started, pass the configuration file to watchtower:

```bash
docker run [...] -v <PATH_TO_HOME_DIR>/.docker/config.json:/config.json patbaumgartner/watchtower:latest
```

When creating the watchtower container via docker-compose, use the following lines:

```yaml
version: "3.4"
services:
  watchtower:
    image: patbaumgartner/watchtower:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - <PATH_TO_HOME_DIR>/.docker/config.json:/config.json
  ...
```

#### Docker Config path
By default, watchtower will look for the `config.json` file in `/`, but this can be changed by setting the `DOCKER_CONFIG` environment variable to the directory path where your config is located. This is useful for setups where the config.json file is changed while the watchtower instance is running, as the changes will not be picked up for a mounted file if the inode changes.
Example usage:

```yaml
version: "3.4"

services: 
  watchtower:
    image: patbaumgartner/watchtower:latest
    environment:
        DOCKER_CONFIG: /config
    volumes:
      - /etc/watchtower/config/:/config/
      - /var/run/docker.sock:/var/run/docker.sock
```

## Credential helpers
Some registries, notably AWS ECR, use a Docker credential helper. Helpers are intentionally not bundled in the
Watchtower image. Install the helper for your platform from its maintained upstream project, then make both the helper
binary and Docker configuration available inside the container.

For [amazon-ecr-credential-helper](https://github.com/awslabs/amazon-ecr-credential-helper), configure the registry in
`config.json`:

```json
{
  "credHelpers": {
    "<AWS_ACCOUNT_ID>.dkr.ecr.<AWS_ECR_REGION>.amazonaws.com": "ecr-login"
  }
}
```

Mount that file at `/config.json`, mount the directory containing `docker-credential-ecr-login`, and include that
directory in `PATH`:

```yaml
services:
  watchtower:
    image: patbaumgartner/watchtower:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./.docker/config.json:/config.json:ro
      - ./bin:/credential-helpers:ro
    environment:
      PATH: /credential-helpers
      AWS_REGION: us-west-1
      DOCKER_API_VERSION: "1.42"
```

Use an EC2 instance role where possible. If static AWS credentials are unavoidable, supply them through a secret
manager rather than committing them to Compose or `config.json`.
