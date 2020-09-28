<?php declare(strict_types=1);
namespace Nais\Device\Exception;

use RuntimeException;

class MissingKolideDeviceException extends RuntimeException implements HealthCheckerException {}