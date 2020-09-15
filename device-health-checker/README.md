# Check and update health status of Nais devices

Scripts dealing with device health status of all Nais devices based on checks from the Kolide API.

## Installation

For development purposes you can clone the repository and install required dependencies:

    git clone git@github.com:nais/device.git
    cd device/device-health-checker
    composer install

Remember to run tests after making changes:

    composer run test

For both of these commands to work you will need to install [Composer](https://getcomposer.org/doc/00-intro.md#installation-linux-unix-macos).

## Releases

A [Phar](https://www.php.net/manual/en/intro.phar.php) archive is built to ease the usage/installation of the scripts in this library. The following archive is generated and [released](https://github.com/nais/device/releases):

- `device-health-checker.phar`

It can be executed like a regular binary once it is set as executable (`chmod +x device-health-checker.phar`).

## Commands

### `kolide:validate-checks`

This command will validate that there exists a criticality level for all Kolide checks:

    ./device-health-checker.phar kolide:validate-checks -t $KOLIDE_API_TOKEN

This command is run as a scheduled workflow in this repository, and if the command finds checks with missing tags it will send a message to the `#nais-device-alarms` channel. This is done using a webhook that is owned by the `Kolide checks validation` Slack app installed on the NAV IT workspace.

#### Command options

##### `-t/--kolide-api-token <token>` (required)

The command must have a working API token to be able to communicate with Kolide.

### `kolide:list-checks`

This command will list all checks that is used with our account on Kolide in JSON format:

    ./device-health-checker.phar kolide:list-checks -t $KOLIDE_API_TOKEN | json_pp

#### Command options

##### `-t/--kolide-api-token <token>` (required)

The command must have a working API token to be able to communicate with Kolide.

### `apiserver:update-devices`

This command is used to update device health status based on live data from the Kolide API.

#### Command options

##### `-t/--kolide-api-token <token>` (required)

The command must have a working API token to be able to communicate with Kolide.

##### `-i/--ignore-checks` (optional, repeatable)

Comma-separated list of Kolide check IDs to ignore when checking device status. For a complete list of checks used with our account use the `kolide:list-checks` command mentioned above.

Some checks are ignored by default (see above), and using the `-i` option will only add checks to the ignore-list. This option can also be repeated, like this:

    -i <id> -i <another id>

##### `-p/--apiserver-password` (required)

Password used for basic auth with the API server.

##### `-u/--apiserver-username` (optional, default: `'device-health-checker'`)

Userame used for Basic auth with the API server. Can be used when testing the script against a local running API server.

#### Environment variables

The script also uses some environment variables:

#### `APISERVER_HOST` (optional, default: `'10.255.240.1'`)

Can be specified to override the default host when communicating with the Nais device API server.

#### `APISERVER_PORT` (optional, default: `''`)

Can be specified to override the default port when communicating with the Nais device API server. If not specified the API client ends up using port `80`.

#### Usage

Simply trigger the script to make it run:

    APISERVER_HOST=localhost APISERVER_PORT=8080 ./device-health-checker.phar apiserver:update-devices -t $KOLIDE_API_TOKEN -p $APISERVER_PASSWORD

During the execution it will output device specific log messages in the following format:

```json
{
    "component": "device-health-checker",
    "system": "nais-device",
    "message": "<log message>",
    "serial": "<device serial>",
    "platform": "<device platform>",
    "username": "<nav email address>",
    "level": "info",
    "timestamp": 1587368677
}
```

For generic log messages the `serial`, `platform` and `username` keys will be omitted. The value of the `timestamp` key is a [Unix timestamp](https://en.wikipedia.org/wiki/Unix_time).

On failure it will output an error message and the exit code will be non-zero.