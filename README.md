# naisdevice

naisdevice is a mechanism enabling NAVs developers to connect to internal resources in a secure and friendly manner.

Each resource is _protected_ by a gateway, and the developer is only granted access to the gateway if all of the following requirements are met:

- Has a valid account
- Has accepted naisdevice [terms and conditions](https://naisdevice-approval.external.prod-gcp.nav.cloud.nais.io/)
- Device is [healthy](#what-is-a-healthy-device)
- Is member of the AAD access group for the gateway (e.g. to connect to team A's DB (via gateway), you must be member of team A's AAD-group)

## Deploying client changes

Executing `make release-frontend` is required for deploy of new naisdevice client to be released and made available for download/install/update.

## key attributes

- minimal attack surface
- instantly reacting to relevant security events
- improved auditlogs: who connected when and to what
- moving away from traditional device management enables building a strong security culture through educating our users on client security instead of automatically configuring their computers

### components

## apiserver

The `apiserver` component serves as the gRPC API server, responsible for handling various configurations and managing communication with other agents. Its primary functionalities include:

- Serving the gRPC API.
- Distributing configurations to the following agents:
  - [device-agent](#device-agent)
  - [gateway-agent](#gateway-agent)
  - [prometheus-agent](#prometheus-agent)
- Retrieving device health status from the `nais/kolide-event-handler`.

## gateway-agent

The `gateway-agent` runs on virtual machines (VMs) and interacts with the `apiserver` to receive and apply configurations. Key features of the `gateway-agent` include:

- Streaming configurations from the `apiserver`.
- Dynamic setup of:
  - WireGuard for communication from devices.
  - iptables for forwarding traffic.

## auth-server

The `auth-server` operates in a cloud run environment and plays a crucial role in user authentication. Its functionalities include:

- Authenticating users.
- Issuing tokens to devices for secure communication.

## enroller

The `enroller` is deployed on Cloud Run and is responsible for managing the enrollment process for both gateways and devices.

- Handling the enrollment of gateways and devices securely.

## device-helper

The `device-helper` serves as the gRPC API for the `device-agent` and performs essential setup tasks for devices. Key functionalities include:

- Providing a gRPC API for the `device-agent`.
- Reading device serial information.
- Configuring network interfaces, routes, and WireGuard for secure communication.

## device-agent

The `device-agent` is a crucial component responsible for managing device configurations and facilitating communication with the `apiserver`. Its main features include:

- Streaming configurations from the `apiserver`.
- Delegating configuration tasks to the `device-helper` via its gRPC API.
- Serving status updates through its gRPC API to the CLI/systray.
- Executing the authentication flow to obtain user tokens.

## systray

The `systray` component acts as a graphical user interface (GUI) for the `agent`, utilizing its gRPC API. It provides a convenient way for users to interact with and monitor the agent's status.

## controlplane-cli

The `controlplane-cli` serves as an administrative command-line interface (CLI) interacting with the `apiserver` through its gRPC API. This CLI is designed for administrative tasks and configurations.

## prometheus-agent

The `prometheus-agent` component connects to all gateways over WireGuard and configures Prometheus (deployed on the same VM) to scrape relevant metrics.

- Establishing connections to gateways using WireGuard.
- Configuring Prometheus to scrape metrics from connected gateways.

## FAQ

### How to install

See https://doc.nais.io/operate/naisdevice/how-to/install/

## Stuff we use

[Kolide](https://www.kolide.com/)

[WireGuard](https://www.wireguard.com)
