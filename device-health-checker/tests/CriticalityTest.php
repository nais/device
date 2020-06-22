<?php declare(strict_types=1);
namespace Nais\Device;

use PHPUnit\Framework\TestCase;

/**
 * @coversDefaultClass Nais\Device\Criticality
 */
class CriticalityTest extends TestCase {
    public function getTags() : array {
        return [
            'valid tag' => [
                'CRITICAL',
                true,
            ],
            'valid tag (lowercase)' => [
                'CRITICAL',
                true,
            ],
            'invalid tag (lowercase)' => [
                'HIGH',
                false,
            ],
        ];
    }

    /**
     * @dataProvider getTags
     * @covers ::isSeverityTag
     */
    public function testCheckSeverityTagsTags(string $tag, bool $isValid) : void {
        $this->assertSame($isValid, Criticality::isSeverityTag($tag), 'Unable to get tag validity');
    }

    public function getTagsForGraceTime() : array {
        return [
            'no tags' => [
                'tags' => [],
                'expectedTime' => Criticality::WARNING,
            ],
            'multiple tags' => [
                'tags' => [
                    'CRITICAL',
                    'LINUX',
                    'WINDOWS'
                ],
                'expectedTime' => Criticality::CRITICAL,
            ],
            'multiple tags including INFO' => [
                'tags' => [
                    'CRITICAL',
                    'LINUX',
                    'INFO'
                ],
                'expectedTime' => Criticality::INFO,
            ],
        ];
    }

    /**
     * @dataProvider getTagsForGraceTime
     * @covers ::getGraceTime
     */
    public function testCanGetTagGraceTime(array $tags, int $expectedTime) : void {
        $this->assertSame($expectedTime, Criticality::getGraceTime($tags));
    }
}