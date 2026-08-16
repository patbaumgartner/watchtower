Watchtower provides an HTTP API mode that enables an HTTP endpoint that can be requested to trigger container updating. The current available endpoint list is:

-   `/v1/update` - triggers an update for all of the containers monitored by this Watchtower instance.

---

To enable this mode, use the flag `--http-api-update`. For example, in a Docker Compose config file:

```yaml
version: '3'

services:
  app-monitored-by-watchtower:
    image: myapps/monitored-by-watchtower
    labels:
      - "com.centurylinklabs.watchtower.enable=true"

  watchtower:
    image: patbaumgartner/watchtower:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    command: --debug --http-api-update
    environment:
      - WATCHTOWER_HTTP_API_TOKEN=mytoken
    labels:
      - "com.centurylinklabs.watchtower.enable=false"
    ports:
      - 127.0.0.1:8080:8080
```

By default, enabling this mode prevents periodic polls (i.e. what is specified using `--interval` or `--schedule`). To run periodic updates regardless, pass `--http-api-periodic-polls`.

Notice that there is an environment variable named `WATCHTOWER_HTTP_API_TOKEN`. All requests must include its value as
a bearer token in the `Authorization` header. The example binds the port to loopback so only the Docker host can reach
it. For remote access, put the endpoint behind a TLS reverse proxy; otherwise the token is sent over cleartext HTTP.
The following `curl` command would trigger an image update:

```bash
curl -H "Authorization: Bearer mytoken" localhost:8080/v1/update
```

---

In order to update only certain images, the image names can be provided as URL query parameters. The following `curl` command would trigger an update for the images `foo/bar` and `foo/baz`:

```bash
curl -H "Authorization: Bearer mytoken" localhost:8080/v1/update?image=foo/bar,foo/baz
```
