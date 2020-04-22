<?php declare(strict_types=1);
namespace Nais;

require 'vendor/autoload.php';

$checks = (new KolideApiClient($_SERVER['KOLIDE_API_TOKEN']))->getAllChecks();
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