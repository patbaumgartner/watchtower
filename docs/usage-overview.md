Watchtower is itself packaged as a Docker container, so installation is as simple as pulling the
`patbaumgartner/watchtower:main` image from Docker Hub. Every tag is a multi-architecture manifest, so `docker pull` and
`docker run` select the build matching your host — there are no per-architecture tags to choose between.

| Platform | Typical hardware |
| --- | --- |
| `linux/amd64` | Intel / AMD servers, desktops, and x86 NAS devices |
| `linux/arm64` | Raspberry Pi 3/4/5 (64-bit OS), Apple Silicon, ARM NAS devices |
| `linux/arm/v7` | Raspberry Pi 2/3 (32-bit OS), older ARM NAS devices |
| `linux/386` | 32-bit x86 hosts |

Images are published to Docker Hub:

```bash
docker pull patbaumgartner/watchtower:main
```

Since the watchtower code needs to interact with the Docker API in order to monitor the running containers, you need to mount _/var/run/docker.sock_ into the container with the `-v` flag when you run it.

Run the `watchtower` container with the following command:

```bash
docker run -d \
  --name watchtower \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e DOCKER_API_VERSION=1.25 \
  patbaumgartner/watchtower:main
```

## Synology and other NAS devices

Watchtower supports Synology DSM 7 via **Container Manager** when its Docker daemon exposes API 1.25 or newer.
API 1.25 is intentionally the default because newer Moby clients no longer support the older daemon versions commonly
shipped on NAS devices.

1. In Container Manager, open **Registry**, search for `watchtower`, and download
    `patbaumgartner/watchtower:main`. Container Manager pulls the manifest entry for your NAS CPU automatically.
2. Create a container from the image and add a bind mount from `/var/run/docker.sock` to `/var/run/docker.sock`.
3. Enable auto-restart so watchtower comes back after a DSM reboot.

Alternatively, use the **Project** feature with this `docker-compose.yml`:

```yaml
services:
  watchtower:
    image: patbaumgartner/watchtower:main
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - TZ=Europe/Zurich
      - WATCHTOWER_CLEANUP=true
      - WATCHTOWER_LABEL_ENABLE=true
      - WATCHTOWER_SCHEDULE=0 0 4 * * *
      - DOCKER_API_VERSION=1.25
```

!!! note "DSM package updates are not managed"
    Watchtower updates containers only. DSM itself and Synology packages are updated through DSM, not watchtower.

!!! warning "Do not let watchtower restart Container Manager's own containers blindly"
    On a NAS it is usually better to opt in explicitly. Start watchtower with `--label-enable` and set
    `com.centurylinklabs.watchtower.enable=true` on the containers you actually want updated. See
    [Container selection](container-selection.md).

If pulling images from private Docker registries, supply registry authentication credentials with the environment variables `REPO_USER` and `REPO_PASS`
or by mounting the host's docker config file into the container (at the root of the container filesystem `/`).

Passing environment variables:

```bash
docker run -d \
  --name watchtower \
  -e REPO_USER=username \
  -e REPO_PASS=password \
  -v /var/run/docker.sock:/var/run/docker.sock \
  patbaumgartner/watchtower:main container_to_watch --debug
```

Also check out [this Stack Overflow answer](https://stackoverflow.com/a/30494145/7872793) for more options on how to pass environment variables.

Alternatively if you 2FA authentication setup on Docker Hub then passing username and password will be insufficient.  Instead you can run `docker login` to store your credentials in `$HOME/.docker/config.json` and then mount this config file to make it available to the Watchtower container:

```bash
docker run -d \
  --name watchtower \
  -v $HOME/.docker/config.json:/config.json \
  -v /var/run/docker.sock:/var/run/docker.sock \
  patbaumgartner/watchtower:main container_to_watch --debug
```

!!! note "Changes to config.json while running"
    If you mount `config.json` in the manner above, changes from the host system will (generally) not be propagated to the
    running container. Mounting files into the Docker daemon uses bind mounts, which are based on inodes. Most
    applications (including `docker login` and `vim`) will not directly edit the file, but instead make a copy and replace
    the original file, which results in a new inode which in turn _breaks_ the bind mount.  
    **As a workaround**, you can create a symlink to your `config.json` file and then mount the symlink in the container. 
    The symlinked file will always have the same inode, which keeps the bind mount intact and will ensure changes
    to the original file are propagated to the running container (regardless of the inode of the source file!).

If you mount the config file as described above, be sure to also prepend the URL for the registry when starting up your
watched image (you can omit the https://). Here is a complete docker-compose.yml file that starts up a docker container
from a private repo on the GitHub Registry and monitors it with watchtower. Note the command argument changing the interval
to 30s rather than the default 24 hours.

```yaml
version: "3"
services:
  cavo:
    image: ghcr.io/<org>/<image>:<tag>
    ports:
      - "443:3443"
      - "80:3080"
  watchtower:
    image: patbaumgartner/watchtower:main
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /root/.docker/config.json:/config.json
    command: --interval 30
```
