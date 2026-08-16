<p style="text-align: center; margin-left: 1.6rem;">
  <img alt="Logotype depicting a lighthouse" src="./images/logo-450px.png" width="450" />
</p>
<h1 align="center">
  Watchtower
</h1>

<p align="center">
  A container-based solution for automating Docker container base image updates.
  <br/><br/>
  <a href="https://github.com/patbaumgartner/watchtower/actions/workflows/ci.yml">
    <img alt="Continuous Integration" src="https://github.com/patbaumgartner/watchtower/actions/workflows/ci.yml/badge.svg?branch=main" />
  </a>
  <a href="https://pkg.go.dev/github.com/patbaumgartner/watchtower">
    <img alt="Go Reference" src="https://pkg.go.dev/badge/github.com/patbaumgartner/watchtower.svg" />
  </a>
  <a href="https://goreportcard.com/report/github.com/patbaumgartner/watchtower">
    <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/patbaumgartner/watchtower" />
  </a>
  <a href="https://github.com/patbaumgartner/watchtower/releases">
    <img alt="latest version" src="https://img.shields.io/github/v/tag/patbaumgartner/watchtower?label=release" />
  </a>
  <a href="https://www.apache.org/licenses/LICENSE-2.0">
    <img alt="Apache-2.0 License" src="https://img.shields.io/github/license/patbaumgartner/watchtower.svg" />
  </a>
</p>

!!! info "Maintained fork"
    This is a maintained fork of [containrrr/watchtower](https://github.com/containrrr/watchtower), which was retired
  upstream. Stable images are published to Docker Hub as `patbaumgartner/watchtower:latest`.

## Quick Start

With watchtower you can update the running version of your containerized app simply by pushing a new image to the Docker
Hub or your own image registry. Watchtower will pull down your new image, gracefully shut down your existing container
and restart it with the same options that were used when it was deployed initially. Run the watchtower container with
the following command:

=== "docker run"

    ```bash
    $ docker run -d \
    --name watchtower \
    -v /var/run/docker.sock:/var/run/docker.sock \
    patbaumgartner/watchtower:latest
    ```

=== "docker-compose.yml"

    ```yaml
    version: "3"
    services:
      watchtower:
        image: patbaumgartner/watchtower:latest
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock
    ```
