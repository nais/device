<?php declare(strict_types=1);
namespace Nais\Device\Command;

use Nais\Device\KolideApiClient;
use PHPUnit\Framework\TestCase;
use RuntimeException;
use Symfony\Component\Console\Tester\CommandTester;

/**
 * @coversDefaultClass Nais\Device\Command\ListChecks
 */
class ListChecksTest extends TestCase {
    /**
     * @covers ::initialize
     */
    public function testFailsOnMissingOption() : void {
        $commandTester = new CommandTester(new ListChecks());
        $this->expectExceptionObject(new RuntimeException(
            'Specity a token for the Kolide API using -t/--kolide-api-token'
        ));
        $commandTester->execute([]);
    }

    public function getChecks() : array {
        return [
            'no checks' => [
                [],
                '[]'
            ],
            'checks' => [
                [
                    [
                        'id'                    => 1,
                        'failing_device_count'  => 0,
                        'name'                  => 'check1',
                        'description'           => 'description1',
                        'notification_strategy' => 'strategy1',
                    ],
                    [
                        'id'                    => 2,
                        'failing_device_count'  => 3,
                        'name'                  => 'check2',
                        'description'           => 'description2',
                        'notification_strategy' => 'strategy2',
                    ],
                ],
                '[{"id":1,"failing_device_count":0,"name":"check1","description":"description1","notification_strategy":"strategy1"},{"id":2,"failing_device_count":3,"name":"check2","description":"description2","notification_strategy":"strategy2"}]'
            ],
        ];
    }

    /**
     * @dataProvider getChecks
     * @covers ::execute
     * @covers ::setKolideApiClient
     * @covers ::initialize
     * @covers ::configure
     * @covers ::__construct
     */
    public function testCanListChecks(array $checks, string $expectedOutput) : void {
        $command = new ListChecks();
        $command->setKolideApiClient($this->createConfiguredMock(KolideApiClient::class, [
            'getAllChecks' => $checks,
        ]));

        $commandTester = new CommandTester($command);
        $commandTester->execute([
            '--kolide-api-token' => 'sometoken',
        ]);

        $output = $commandTester->getDisplay();

        $this->assertSame($expectedOutput, trim($output));
        $this->assertSame(0, $commandTester->getStatusCode(), 'Expected command to return 0');
    }
}