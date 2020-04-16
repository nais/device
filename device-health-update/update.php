<?php declare(strict_types=1);
namespace Nais;

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

$kolideChecksBlacklist = !empty($_SERVER['KOLIDE_CHECKS_BLACKLIST'])
    ? array_map(function(string $id) : int { return (int) trim($id); }, explode(',', $_SERVER['KOLIDE_CHECKS_BLACKLIST']))
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

// Fetch serials for all devices that is currently failing according to Kolide (ignoring all
// blacklisted checks).
$failingKolideDevices = array_column($kolideApiClient->getFailingDevices($kolideChecksBlacklist), 'serial');

// Fetch all current Nais devices, and make sure to set the isHealthy flag to false if the device
// seems to be failing according to Kolide. All other devices will be set to healthy.
$updatedNaisDevices = array_map(function(array $naisDevice) use ($failingKolideDevices) : array {
    return [
        'serial' => $naisDevice['serial'],
        'isHealthy' => !in_array($naisDevice['serial'], $failingKolideDevices),
    ];
}, json_decode($naisDeviceApiClient->get('/devices')->getBody()->getContents(), true));

// Trigger the actual update of the devices.
$naisDeviceApiClient->put('/devices/health', ['json' => $updatedNaisDevices]);