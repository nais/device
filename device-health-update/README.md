# Update health status of Nais devices

Script that will update health status of all Nais devices based on checks from the Kolide.

## Installation

Clone the repository and install required dependencies:

    git clone git@github.com:nais/device.git
    cd device/device-health-update
    composer install --no-dev

If you wish to install development dependencies as well (to for instance run the test suite locally), skip the `--no-dev` flag above.

## Supported environment variables

### `KOLIDE_API_TOKEN` (required)

Used for authentication with the Kolide API.

### `KOLIDE_CHECKS_BLACKLIST` (optional, default: `''`)

Comma-separated list of Kolide check IDs to ignore when checking device status. For a complete list of checks used with our account run the following script:

    php get-checks.php

The above command requires the `KOLIDE_API_TOKEN` environment variable to be able to communicate with the Kolide API.

### `APISERVER_HOST` (optional, default: `'apiserver.device.nais.io'`)

Can be specified to override the default host when communicating with the Nais device API server.

### `APISERVER_PORT` (optional, default: `''`)

Can be specified to override the default port when communicating with the Nais device API server. If not specified the API client ends up using port `80`.

## Usage

The script that updates device statuses is executed in the following way:

    php update.php

On failure it will output an error message and the exit code will be non-zero.