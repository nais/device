<?php declare(strict_types=1);
namespace Nais\Device;

class Severity {
    const INFO     = -1;     // Used for checks we want to ignore
    const NOTICE   = 604800; // 7 days
    const WARNING  = 172800; // 2 days
    const DANGER   = 3600;   // 1 hour
    const CRITICAL = 0;      // Not allowed

    /**
     * Valid severity tags
     *
     * @var array<string, int>
     */
    static $tags = [
        'info'     => self::INFO,
        'notice'   => self::NOTICE,
        'warning'  => self::WARNING,
        'danger'   => self::DANGER,
        'critical' => self::CRITICAL,
    ];

    /**
     * Get the grace time given a set of tags
     *
     * @param string[] $tags
     * @return int
     */
    public static function getGraceTime(array $tags) : int {
        $severityLevels = [self::WARNING];

        foreach ($tags as $tag) {
            $severityLevels[] = self::$tags[strtolower($tag)] ?? self::WARNING;
        }

        return min($severityLevels);
    }

    /**
     * Check if a tag is a severity tag
     *
     * @param string $tag
     * @return bool
     */
    public static function isSeverityTag(string $tag) : bool {
        return in_array(strtolower($tag), array_keys(self::$tags));
    }
}