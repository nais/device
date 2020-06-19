<?php declare(strict_types=1);
namespace Nais\Device;

class Criticality {
    const IGNORE = -1;     // Used for checks we want to ignore
    const LOW    = 604800; // 7 days
    const MED    = 172800; // 2 days
    const HIGH   = 3600;   // 1 hour
    const CRIT   = 0;      // Not allowed

    const INFO     = -1;     // Used for checks we want to ignore
    const NOTICE   = 604800; // 7 days
    const WARNING  = 172800; // 2 days
    const DANGER   = 3600;   // 1 hour
    const CRITICAL = 0;      // Not allowed

    static $tags = [
        'info'     => self::INFO,
        'notice'   => self::NOTICE,
        'warning'  => self::WARNING,
        'danger'   => self::DANGER,
        'critical' => self::CRITICAL,
    ];

    public static function getTagGraceTime(string $tag) : int {
        return self::$tags[strtolower($tag)] ?? 172800;
    }

    public static function isValidTag(string $tag) : bool {
        return in_array(strtolower($tag), array_keys(self::$tags));
    }
}