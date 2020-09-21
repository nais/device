# naisdevice

naisdevice is a mechanism enabling NAVs developers to connect to internal resources in a secure and friendly manner. 

Each resource is protected by a gateway, and the developer is only granted access to the gateway if all of the following requirements are met:
- Has a valid nav.no account
- Has accepted naisdevice [terms and conditions](https://naisdevice-approval.nais.io/)
- Device is [healthy](#what-is-a-healthy-device)
- Is member of the AAD access group for the gateway (e.g. to connect to team A's DB, you must be member of team A's AAD-group)

## key attributes

- minimal attack surface
- frequent key rotation
- instantly reacting to relevant security events
- improved auditlogs: who connected when and to what, as well as other relevant user events
- moving away from traditional device management enables building a strong security culture through educating our users on client security instead of automatically configuring their computers

## architecture

todo: simple visual describing:
- apiserver coordinates configuration
- device + gateway fetches config on a timer
- device-health-checker informs apiserver of device health from Kolide
- additionally: bootstrap-api used first time user connects/enrolls into the system

### components 

#### apiserver
The naisdevice apiserver main responsibility is to serve the [device-agents](#device-agent) and [gateway-agents](#gateway-agent) with configuration through a set of APIs.

It's database is master for all peers (devices and gateways) operating in the environment, as well as keeping track of and allocating IPs in the VPN's address space.

It calculates the appropriate configuration for the peers primarily based on two factors:
1. Is the device owner authorized to use the gateway?
2. Is the device in a healthy state?

If both is true, the device-agent and gateway-agent is informed with the necessary information in order for them to communicate. 

The apiserver also:
- exposes an endpoint for other trusted processes to inform about the health status of the devices. Currently this is the [device-health-checker](#device-health-checker) components
- continuously checks the [bootstrap-api](#bootstrap-api) for any pending enrollments requests and adds new devices.

### device-agent
### gateway-agent
### device-health-checker
### bootstrap-api

## [Kolide](https://www.kolide.com/)

## [WireGuard](https://www.wireguard.com)

## FAQ
### What is a healthy device?
### How to install
See https://doc.nais.io/device
