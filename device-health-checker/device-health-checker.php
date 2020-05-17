<?php
namespace Nais\Device;

use Symfony\Component\Console\Application;
use Throwable;

set_exception_handler(function(Throwable $e) : void {
    echo json_encode([
        'component' => 'device-health-checker',
        'system'    => 'nais-device',
        'message'   => $e->getMessage(),
        'timestamp' => time(),
    ]) . PHP_EOL;
    exit(1);
});

require 'vendor/autoload.php';

$checksCriticality = require 'checks-criticality.php';

$application = new Application('Device health checker');
$application->add(new Command\ListChecks());
$application->add(new Command\CheckAndUpdateDevices($checksCriticality));
$application->run();
