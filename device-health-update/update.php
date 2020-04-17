<?php declare(strict_types=1);
namespace Nais;

use DateTime;
use GuzzleHttp\Client as HttpClient;
use RuntimeException;
use Throwable;

require 'vendor/autoload.php';

set_exception_handler(function(Throwable $e) : void {
    echo trim($e->getMessage()) . PHP_EOL;
    exit($e->getCode());
});

foreach (['KOLIDE_API_TOKEN'] as $requiredEnvVar) {
    if (empty($_SERVER[$requiredEnvVar])) {
        throw new RuntimeException(sprintf('Missing required environment variable: %s', $requiredEnvVar), 1);
    }
}

$kolideChecksIgnored = !empty($_SERVER['KOLIDE_CHECKS_IGNORED'])
    ? array_map(function(string $id) : int { return (int) trim($id); }, explode(',', $_SERVER['KOLIDE_CHECKS_IGNORED']))
    : [];

$schema = 'http';
$host   = $_SERVER['APISERVER_HOST'] ?? 'apiserver.device.nais.io';
$port   = $_SERVER['APISERVER_PORT'] ?? '';

if (443 == $port) {
    $schema = 'https';
    $port = '';
}

$naisDeviceApiClient = new HttpClient(['base_uri' => trim(sprintf('%s://%s:%s', $schema, $host, $port), ':')]);
$kolideApiClient = new KolideApiClient($_SERVER['KOLIDE_API_TOKEN']);

// Fetch all failing checks from the Kolide API
$failingChecks = $kolideApiClient->getFailingChecks($kolideChecksIgnored);

// Devices serials that will be marked as unhealthy
$failingDeviceSerials = [];

// Can be removed once Kolide includes criticality in the API
$checksConfig = require 'checks-config.php';

// When check is missing from the config fetched above (can occur if Kolide introduces a new
// check). Can also be removed once Kolide gets criticality in the API response.
$defaultCriticality = Criticality::MED;

// Get current timestamp that will be used to check against the criticality of the failing check
$now = (new DateTime('now'))->getTimestamp();

foreach ($failingChecks as $check) {
    $criticality = $checksConfig[$check['id']] ?? $defaultCriticality;
    $failures = $kolideApiClient->getCheckFailures($check['id']);

    foreach ($failures as $failure) {
        // Ignore the device if Kolide does not have the serial
        if ('-1' === $failure['device']['serial']) {
            continue;
        }

        // Failure has been resolved, skip this one
        if (null !== $failure['resolved_at']) {
            continue;
        }

        $occurredAt = (new DateTime($failure['timestamp']))->getTimestamp();

        // If the diff in seconds is above the current criticality level the device will be marked
        // as unhealthy
        if (($now - $occurredAt) > $criticality) {
            $failingDeviceSerials[] = $failure['device']['serial'];
        }
    }
}

// Remove duplicates
$failingDeviceSerials = array_unique($failingDeviceSerials);

// Fetch all current Nais devices, and make sure to set the isHealthy flag to false if the device
// seems to be failing according to Kolide. All other devices will be set to healthy.
$updatedNaisDevices = array_map(function(array $naisDevice) use ($failingDeviceSerials) : array {
    return [
        'serial' => $naisDevice['serial'],
        'isHealthy' => !in_array($naisDevice['serial'], $failingDeviceSerials),
    ];
}, json_decode($naisDeviceApiClient->get('/devices')->getBody()->getContents(), true));

// Trigger the actual update of the devices.
$naisDeviceApiClient->put('/devices/health', ['json' => $updatedNaisDevices]);