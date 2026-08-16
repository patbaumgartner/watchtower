By default, watchtower is set-up to monitor the local Docker daemon (the same daemon running the watchtower container itself). However, it is possible to configure watchtower to monitor a remote Docker endpoint. When starting the watchtower container you can specify a remote Docker endpoint with either the `--host` flag or the `DOCKER_HOST` environment variable:

```bash
docker run -d \
  --name watchtower \
  patbaumgartner/watchtower:latest --host "tcp://10.0.1.2:2375"
```

or

```bash
docker run -d \
  --name watchtower \
  -e DOCKER_HOST="tcp://10.0.1.2:2375" \
  patbaumgartner/watchtower:latest
```

Note in both of the examples above that it is unnecessary to mount the _/var/run/docker.sock_ into the watchtower container.

For a TLS-protected daemon, also pass `DOCKER_TLS_VERIFY=1` and mount the client certificates into the container. The
Docker SDK reads the certificate directory from `DOCKER_CERT_PATH` and registry credentials from `DOCKER_CONFIG`.
Never expose an unauthenticated Docker TCP socket; control of the Docker API is equivalent to root access on the host.
