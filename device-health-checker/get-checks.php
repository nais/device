<?php declare(strict_types=1);
namespace Nais;

use GuzzleHttp\Exception\ClientException;

require 'vendor/autoload.php';

foreach (['KOLIDE_API_TOKEN'] as $requiredEnvVar) {
    if (empty($_SERVER[$requiredEnvVar])) {
        echo sprintf('Missing required environment variable: %s', $requiredEnvVar) . PHP_EOL;
        exit(1);
    }
}

try {
    $checks = (new KolideApiClient($_SERVER['KOLIDE_API_TOKEN']))->getAllChecks();
} catch (ClientException $e) {
    echo $e->getMessage();
    exit($e->getCode());
}
$maxIdLength = strlen((string) max(array_column($checks, 'id')));
$maxNameLength = max(array_map('strlen', array_column($checks, 'name')));

echo sprintf(
    '%s | %s | URL',
    str_pad('ID', $maxIdLength, ' '),
    str_pad('Name', $maxNameLength, ' ')
) . PHP_EOL;

foreach ($checks as $check) {
    echo sprintf(
        '%s | %s | https://k2.kolide.com/1401/checks/%d',
        str_pad((string) $check['id'], $maxIdLength, ' '),
        str_pad($check['name'], $maxNameLength, ' '),
        $check['id']
    ) . PHP_EOL;
}