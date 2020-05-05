# Check and update health status of Nais devices

Scripts dealing with device health status of all Nais devices based on checks from the Kolide API.

## Installation

For development purposes you can clone the repository and install required dependencies:

    git clone git@github.com:nais/device.git
    cd device/device-health-checker
    composer install

Remember to run tests after making changes:

    composer run ci

## Releases

[Phar](https://www.php.net/manual/en/intro.phar.php) archives are built to ease the usage/installation of the scripts in this library. The following archives are generated and [released](https://github.com/nais/device/releases):

- `get-checks.phar`
- `update.phar`

The archives are also built and uploaded as artifacts to the [Build and test device health update](https://github.com/nais/device/actions?query=workflow%3A%22Build+and+test+device+health+update%22) workflow.

They can be executed like binaries once they are set as executable.

## Script: `get-checks.phar`

This script is used to display all checks connected to our account on Kolide.

### Supported environment variables

#### `KOLIDE_API_TOKEN` (required)

Used for authentication with the Kolide API.

### Usage

```
christer_edvartsen@apiserver:~$ ./get-checks.phar
ID    | Name                                 | URL
32853 | macOS Secure Keyboard Entry Disabled | https://k2.kolide.com/1401/checks/32853
32837 | File Extensions Not Visible To User  | https://k2.kolide.com/1401/checks/32837
32834 | Unencrypted SSH Key                  | https://k2.kolide.com/1401/checks/32834
32836 | Find My Mac Disabled                 | https://k2.kolide.com/1401/checks/32836
...
```

## Script: `update.phar`

This script is used to update device health status based on live data from the Kolide API.

### Supported environment variables

#### `KOLIDE_API_TOKEN` (required)

Used for authentication with the Kolide API.

#### `KOLIDE_CHECKS_IGNORED` (optional, default: `''`)

Comma-separated list of Kolide check IDs to ignore when checking device status. For a complete list of checks used with our account use the `get-checks.phar` script mentioned above.

#### `APISERVER_PASSWORD` (required, default: `''`)

Password needed when authenticating requests to the API server.

#### `APISERVER_HOST` (optional, default: `'10.255.240.1'`)

Can be specified to override the default host when communicating with the Nais device API server.

#### `APISERVER_PORT` (optional, default: `''`)

Can be specified to override the default port when communicating with the Nais device API server. If not specified the API client ends up using port `80`.

### Usage

Simply trigger the script to make it run:

```
christer_edvartsen@apiserver:~$ ./update.phar
...
```

During the execution it will output log message in the following format:

```json
{
    "component": "device-health-checker",
    "system": "nais-device",
    "message": "<log message>",
    "serial": "<device serial>",
    "username": "<nav email address>",
    "level": "info",
    "timestamp": 1587368677
}
```

For generic log messages the `serial` and `username` keys will be omitted. The value of the `timestamp` key is a [Unix timestamp](https://en.wikipedia.org/wiki/Unix_time).

On failure it will output an error message and the exit code will be non-zero.