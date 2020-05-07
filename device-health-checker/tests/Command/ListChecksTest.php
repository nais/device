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
        $checksOutput = <<<OUTPUT
ID     | Name             | URL
====== | ================ | ========================================
123    | some check       | https://k2.kolide.com/1401/checks/123
456789 | some other check | https://k2.kolide.com/1401/checks/456789
OUTPUT;

        return [
            'no checks' => [
                [],
                'No checks exist'
            ],
            'checks' => [
                [
                    [
                        'id'                    => 123,
                        'failing_device_count'  => 0,
                        'name'                  => 'some check',
                        'description'           => 'some description',
                        'notification_strategy' => 'some strategy',
                    ],
                    [
                        'id'                    => 456789,
                        'failing_device_count'  => 3,
                        'name'                  => 'some other check',
                        'description'           => 'some other description',
                        'notification_strategy' => 'some other strategy',
                    ],
                ],
                $checksOutput
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