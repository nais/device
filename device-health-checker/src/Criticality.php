<?php declare(strict_types=1);
namespace Nais\Device;

class Criticality {
    const IGNORE = -1;     // Used for checks we want to ignore
    const LOW    = 604800; // 7 days
    const MED    = 172800; // 2 days
    const HIGH   = 3600;   // 1 hour
}