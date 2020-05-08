<?php
namespace Nais\Device;

require 'vendor/autoload.php';

use Symfony\Component\Console\Application;

$application = new Application('Device health checker');
$application->add(new Command\ListChecks());
$application->add(new Command\CheckAndUpdateDevices(require 'checks-config.php'));
$application->run();