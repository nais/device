<?php declare(strict_types=1);
namespace Nais\Device\Exception;

use RuntimeException;

class MultipleKolideDevicesException extends RuntimeException implements HealthCheckerException {}