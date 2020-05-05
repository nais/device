<?php declare(strict_types=1);
namespace Nais;

use DateTime;
use GuzzleHttp\Client as HttpClient;
use RuntimeException;
use Throwable;

const LOGLEVEL_INFO  = 'info';
const LOGLEVEL_ERROR = 'error';

require 'vendor/autoload.php';

function log(string $message, string $level = LOGLEVEL_INFO, string $serial = null, string $username = null) : void {
    echo json_encode(array_filter([
        'component' => 'device-health-checker',
        'system'    => 'nais-device',
        'message'   => $message,
        'serial'    => $serial,
        'username'  => $username,
        'level'     => $level,
        'timestamp' => time(),
    ])) . PHP_EOL;
}

set_exception_handler(function(Throwable $e) : void {
    log($e->getMessage(), LOGLEVEL_ERROR);
    exit(1);
});

foreach (['KOLIDE_API_TOKEN', 'APISERVER_PASSWORD'] as $requiredEnvVar) {
    if (empty($_SERVER[$requiredEnvVar])) {
        throw new RuntimeException(sprintf('Missing required environment variable: %s', $requiredEnvVar));
    }
}

$kolideChecksIgnored = !empty($_SERVER['KOLIDE_CHECKS_IGNORED'])
    ? array_map(function(string $id) : int { return (int) trim($id); }, explode(',', $_SERVER['KOLIDE_CHECKS_IGNORED']))
    : [];

$schema = 'http';
$host   = $_SERVER['APISERVER_HOST'] ?? '10.255.240.1';
$port   = $_SERVER['APISERVER_PORT'] ?? '';

if (443 == $port) {
    $schema = 'https';
    $port = '';
}

$naisDeviceApiClient = new HttpClient([
    'base_uri' => trim(sprintf('%s://%s:%s', $schema, $host, $port), ':'),
    'timeout'  => 3,
    'auth'     => ['device-health-checker', $_SEVER['APISERVER_PASSWORD']],
]);
$kolideApiClient = new KolideApiClient($_SERVER['KOLIDE_API_TOKEN']);

// Failing devices that will be marked as unhealthy
$failingDevices = [];

// Can be removed once Kolide includes criticality in the API
$checksConfig = require 'checks-config.php';

// When check is missing from the config fetched above (can occur if Kolide introduces a new
// check). Can also be removed once Kolide gets criticality in the API response.
$defaultCriticality = Criticality::MED;

// Fetch all failing checks from the Kolide API
$failingChecks = $kolideApiClient->getFailingChecks($kolideChecksIgnored);

// Get current timestamp that will be used to check against the criticality of the failing check
$now = (new DateTime('now'))->getTimestamp();

foreach ($failingChecks as $check) {
    $criticality = $checksConfig[$check['id']] ?? $defaultCriticality;
    $failures = $kolideApiClient->getCheckFailures($check['id']);

    foreach ($failures as $failure) {
        $serial = $failure['device']['serial'];

        // Ignore the device if Kolide does not have the serial
        if ('-1' === $serial) {
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
            if (!isset($failingDevices[$serial])) {
                $failingDevices[$serial] = [
                    'username' => $failure['device']['assigned_owner']['email'],
                    'failures' => [],
                ];
            }

            $failingDevices[$serial]['failures'][] = $failure;
        }
    }
}

// Fetch all current Nais devices, and make sure to set the isHealthy flag to false if the device
// seems to be failing according to Kolide. All other devices will be set to healthy.
$updatedNaisDevices = array_map(function(array $naisDevice) use ($failingDevices) : array {
    $serial         = $naisDevice['serial'];
    $alreadyHealthy = $naisDevice['isHealthy'];
    $username       = $naisDevice['username'];
    $healthy        = !array_key_exists($serial, $failingDevices);

    if ($healthy && !$alreadyHealthy) {
        log('No failing checks anymore, device is now healthy', LOGLEVEL_INFO, $serial, $username);
    } else if ($alreadyHealthy && !$healthy) {
        $failingChecks = array_map(function(array $failure) : string {
            return $failure['title'];
        }, $failingDevices[$serial]['failures']);
        log(
            sprintf('Device is no longer healthy because of the following failing checks: %s', join(', ', $failingChecks)),
            LOGLEVEL_INFO,
            $serial,
            $username
        );
    }

    return [
        'serial' => $naisDevice['serial'],
        'isHealthy' => $healthy,
    ];
}, json_decode($naisDeviceApiClient->get('/devices')->getBody()->getContents(), true) ?: []);

// Trigger the actual update of the devices.
$naisDeviceApiClient->put('/devices/health', ['json' => $updatedNaisDevices]);
log('Sent updated Nais device configuration to API server');