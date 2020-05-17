<?php declare(strict_types=1);
namespace Nais\Device\Command;

use Nais\Device\Criticality;
use Nais\Device\KolideApiClient;
use PHPUnit\Framework\TestCase;
use Symfony\Component\Console\Tester\CommandTester;

/**
 * @coversDefaultClass Nais\Device\Command\ValidateKolideChecksCriticality
 */
class ValidateKolideChecksCriticalityTest extends TestCase {
    /**
     * @covers ::execute
     * @covers ::__construct
     * @covers ::configure
     * @covers ::initialize
     */
    public function testReturnsZeroWhenNoChecksAreMissing() : void {
        $command = new ValidateKolideChecksCriticality([
            1 => Criticality::HIGH,
            2 => Criticality::IGNORE,
        ]);
        $command->setKolideApiClient($this->createConfiguredMock(KolideApiClient::class, [
            'getAllChecks' => [
                [
                    'id'          => 1,
                    'name'        => 'some name',
                    'description' => 'some description',
                ],
                [
                    'id'          => 2,
                    'name'        => 'some other name',
                    'description' => 'some other description',
                ],
            ],
        ]));

        $commandTester = new CommandTester($command);
        $exitCode = $commandTester->execute([]);

        $display = trim($commandTester->getDisplay());

        $this->assertSame(0, $exitCode, 'Expected exit code to be 0');
        $this->assertSame('All checks have been configured', $display, 'Incorrect display');
    }

    /**
     * @covers ::execute
     * @covers ::__construct
     * @covers ::configure
     * @covers ::initialize
     */
    public function testReturnsNonZeroOnFailure() : void {
        $command = new ValidateKolideChecksCriticality([
            1 => Criticality::HIGH,
        ]);
        $command->setKolideApiClient($this->createConfiguredMock(KolideApiClient::class, [
            'getAllChecks' => [
                [
                    'id'          => 1,
                    'name'        => 'some name',
                    'description' => 'some description',
                ],
                [
                    'id'          => 2,
                    'name'        => 'some other name',
                    'description' => 'some other description',
                ],
                [
                    'id'          => 3,
                    'name'        => 'some third name',
                    'description' => 'some third description',
                ],
            ],
        ]));

        $commandTester = new CommandTester($command);
        $exitCode = $commandTester->execute([]);

        $display = trim($commandTester->getDisplay());
        $expectedDisplay = <<<DISPLAY
The following Kolide checks are missing a criticality level:
some other name (ID: 2, https://k2.kolide.com/1401/checks/2): some other description
some third name (ID: 3, https://k2.kolide.com/1401/checks/3): some third description
DISPLAY;

        $this->assertSame(1, $exitCode, 'Expected exit code to be 1');
        $this->assertSame($expectedDisplay, $display, 'Incorrect display');
    }
}