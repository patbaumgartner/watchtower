# Security Policy

## Supported Versions

Security updates are only applied to the latest release. Because the `latest` tag is what most deployments track,
and watchtower updates itself, those fixes reach you on the next update cycle.

## Reporting a Vulnerability

Report vulnerabilities privately through GitHub:
[**open a security advisory**](https://github.com/patbaumgartner/watchtower/security/advisories/new).
That channel is private until an advisory is published, so please use it rather than a public issue or discussion.

Please include the watchtower version, the platform, and the smallest reproduction you have.

This is a best-effort community fork maintained in spare time, so no response-time guarantee is offered. Expect an
acknowledgement within a week; if you have not heard back by then, feel free to ping the advisory thread.

## Scope

Watchtower requires access to the Docker socket, which is equivalent to root on the host. Findings that amount to
"a process with the Docker socket can control the host" are inherent to the design and out of scope. Findings that
let a *remote* party — a registry, a notification endpoint, or an HTTP API client — influence watchtower are in scope.
