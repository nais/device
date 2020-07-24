<?php declare(strict_types=1);
namespace Nais\Device\Command;

use Nais\Device\ApiServerClient;
use Nais\Device\KolideApiClient;
use PHPUnit\Framework\TestCase;
use Symfony\Component\Console\Application;
use Symfony\Component\Console\Tester\ApplicationTester;
use Symfony\Component\Console\Tester\CommandTester;
use RuntimeException;

/**
 * @coversDefaultClass Nais\Device\Command\CheckAndUpdateDevices
 */
class CheckAndUpdateDevicesTest extends TestCase {
    public function getMissingParams() : array {
        return [
            'missing token' => [
                [
                    '-u' => 'username',
                    '-p' => 'password',
                ],
                'Specity a token',
            ],
            'missing password' => [
                [
                    '-u' => 'username',
                    '-t' => 'token',
                ],
                'Specity a password',
            ],
        ];
    }

    /**
     * @dataProvider getMissingParams
     * @covers ::initialize
     */
    public function testFailsOnMissingOptions(array $params, string $error) : void {
        $commandTester = new CommandTester(new CheckAndUpdateDevices());
        $this->expectExceptionObject(new RuntimeException($error));
        $commandTester->execute($params);
    }

    public function getDeviceData() : array {
        return [
            'no nais devices' => [
                'naisDevices' => [],
                'kolideDevices' => [],
                'expectedMessages' => [
                    'Fetched devices from Kolide',
                    'No Nais devices to update :(',
                ],
                'expectedLogSerials' => [
                    null,
                    null,
                ],
                'expectedLogPlatforms' => [
                    null,
                    null,
                ],
                'expectedLogUsernames' => [
                    null,
                    null,
                ],
                'expectedUpdatePayload' => [],
            ],
            'no matching kolide devices' => [
                'naisDevices' => [
                    [
                        'serial'         => 'serial1',
                        'platform'       => 'linux',
                        'username'       => 'user1@nav.no',
                        'isHealthy'      => true,
                        'kolideLastSeen' => 1589952551,
                    ],
                ],
                'kolideDevices' => [],
                'expectedLogMessages' => [
                    'Fetched devices from Kolide',
                    'Could not find matching Kolide device',
                    'No Nais devices to update :('
                ],
                'expectedLogSerials' => [
                    null,
                    'serial1',
                    null,
                ],
                'expectedLogPlatforms' => [
                    null,
                    'linux',
                    null,
                ],
                'expectedLogUsernames' => [
                    null,
                    'user1@nav.no',
                    null,
                ],
                'expectedUpdatePayload' => [],
            ],
            'no failing checks' => [
                'naisDevices' => [
                    [
                        'serial'         => 'serial1',
                        'platform'       => 'linux',
                        'username'       => 'user1@nav.no',
                        'isHealthy'      => true,
                        'kolideLastSeen' => 1589952551,
                    ],
                    [
                        'serial'         => 'serial2',
                        'platform'       => 'linux',
                        'username'       => 'user2@nav.no',
                        'isHealthy'      => true,
                        'kolideLastSeen' => 1589952551,
                    ],
                ],
                'kolideDevices' => [
                    [
                        'id'                     => 1,
                        'serial'                 => 'serial1',
                        'platform'               => 'ubuntu',
                        'assigned_owner'         => ['email' => 'user1@nav.no'],
                        'failure_count'          => 0,
                        'last_seen_at'           => $this->getTimestamp(1589956055),
                    ],
                    [
                        'id'                     => 2,
                        'serial'                 => 'serial2',
                        'platform'               => 'rhel',
                        'assigned_owner'         => ['email' => 'user2@nav.no'],
                        'failure_count'          => 0,
                        'last_seen_at'           => $this->getTimestamp(1589956055),
                    ],
                ],
                'expectedLogMessages' => [
                    'Fetched devices from Kolide',
                    'Sent updated Nais device configuration to API server'
                ],
                'expectedLogSerials' => [
                    null,
                    null,
                ],
                'expectedLogPlatforms' => [
                    null,
                    null
                ],
                'expectedLogUsernames' => [
                    null,
                    null,
                ],
                'expectedUpdatePayload' => [
                    [
                        'serial'         => 'serial1',
                        'platform'       => 'linux',
                        'username'       => 'user1@nav.no',
                        'isHealthy'      => true,
                        'kolideLastSeen' => 1589956055,
                    ],
                    [
                        'serial'         => 'serial2',
                        'platform'       => 'linux',
                        'username'       => 'user2@nav.no',
                        'isHealthy'      => true,
                        'kolideLastSeen' => 1589956055,
                    ],
                ],
            ],
        ];
    }

    /**
     * @dataProvider getDeviceData
     * @covers ::execute
     * @covers ::identifyKolideDevice
     * @covers ::log
     */
    public function testCanUpdateDevices(array $naisDevices, array $kolideDevices, array $expectedLogMessages, array $expectedLogSerials, array $expectedLogPlatforms, array $expectedLogUsernames, array $expectedUpdatePayload) : void {
        $apiServerClient = $this->createConfiguredMock(ApiServerClient::class, [
            'getDevices' => $naisDevices,
        ]);
        $command = new CheckAndUpdateDevices();
        $command->setApiServerClient($apiServerClient);
        $command->setKolideApiClient($this->createConfiguredMock(KolideApiClient::class, [
            'getAllDevices' => $kolideDevices,
        ]));

        if (!empty($expectedUpdatePayload)) {
            $apiServerClient
                ->expects($this->once())
                ->method('updateDevices')
                ->with($expectedUpdatePayload);
        }

        $application = new Application('Device health checker test');
        $application->setAutoExit(false);
        $application->add($command);

        $applicationTester = new ApplicationTester($application);
        $exitCode = $applicationTester->run([
            'command' => $command->getName(),
        ]);
        $display = explode(PHP_EOL, trim($applicationTester->getDisplay()));

        $this->assertSame(0, $exitCode, 'Expected exit code to be 0');
        $this->assertSame(
            $expectedLogMessages,
            array_map(fn($msg) => json_decode($msg, true)['message'], $display),
            'Unexpected message in logs'
        );
        $this->assertSame(
            $expectedLogSerials,
            array_map(fn($msg) => json_decode($msg, true)['serial'] ?? null, $display),
            'Unexpected serial in logs'
        );
        $this->assertSame(
            $expectedLogPlatforms,
            array_map(fn($msg) => json_decode($msg, true)['platform'] ?? null, $display),
            'Unexpected platform in logs'
        );
        $this->assertSame(
            $expectedLogUsernames,
            array_map(fn($msg) => json_decode($msg, true)['username'] ?? null, $display),
            'Unexpected username in logs'
        );
    }

    /**
     * @covers ::execute
     * @covers ::identifyKolideDevice
     * @covers ::getFailingDeviceChecks
     * @covers ::log
     * @covers ::setApiServerClient
     * @covers ::setKolideApiClient
     * @covers ::initialize
     */
    public function testCanUpdateDevicesWithFailingChecks() : void {
        $apiServerClient = $this->createConfiguredMock(ApiServerClient::class, [
            'getDevices' => [
                // Healthy device with no failing Kolide checks
                [
                    'serial'         => 'serial1',
                    'platform'       => 'darwin',
                    'username'       => 'user1@nav.no',
                    'isHealthy'      => true,
                    'kolideLastSeen' => 1589952551,
                ],
                // Healthy device, with no matching Kolide device
                [
                    'serial'         => 'serial2-not-in-kolide',
                    'platform'       => 'darwin',
                    'username'       => 'user2@nav.no',
                    'isHealthy'      => true,
                    'kolideLastSeen' => 1589952552
                ],
                // Non-healthy device with no failing Kolide checks
                [
                    'serial'         => 'serial3',
                    'platform'       => 'linux',
                    'username'       => 'user3@nav.no',
                    'isHealthy'      => false,
                    'kolideLastSeen' => 1589952553,
                ],
                // Healthy device with failing Kolide check
                [
                    'serial'         => 'serial4',
                    'platform'       => 'linux',
                    'username'       => 'user4@nav.no',
                    'isHealthy'      => true,
                    'kolideLastSeen' => 1589952554,
                ],
                // Healthy device with failing Kolide check, but not above time limit
                [
                    'serial'         => 'serial5',
                    'platform'       => 'linux',
                    'username'       => 'user5@nav.no',
                    'isHealthy'      => true,
                    'kolideLastSeen' => 1589952555,
                ],
            ],
        ]);
        $apiServerClient
            ->expects($this->once())
            ->method('updateDevices')
            ->with([
                [
                    'serial'         => 'serial1',
                    'platform'       => 'darwin',
                    'username'       => 'user1@nav.no',
                    'isHealthy'      => true,
                    'kolideLastSeen' => 1589956403,
                ],
                [
                    'serial'         => 'serial3',
                    'platform'       => 'linux',
                    'username'       => 'user3@nav.no',
                    'isHealthy'      => true,
                    'kolideLastSeen' => 1589956403,
                ],
                [
                    'serial'         => 'serial4',
                    'platform'       => 'linux',
                    'username'       => 'user4@nav.no',
                    'isHealthy'      => false,
                    'kolideLastSeen' => 1589956403,
                ],
                [
                    'serial'         => 'serial5',
                    'platform'       => 'linux',
                    'username'       => 'user5@nav.no',
                    'isHealthy'      => true,
                    'kolideLastSeen' => 1589956403,
                ],
            ]);

        $kolideApiClient = $this->createConfiguredMock(KolideApiClient::class, [
            'getAllDevices' => [
                [
                    'id'                     => 1,
                    'serial'                 => 'serial1',
                    'platform'               => 'darwin',
                    'assigned_owner'         => ['email' => 'user1@nav.no'],
                    'failure_count'          => 0,
                    'last_seen_at'           => $this->getTimestamp(1589956403),
                ],
                [
                    'id'                     => 2,
                    'serial'                 => 'serial3',
                    'platform'               => 'rhel',
                    'assigned_owner'         => ['email' => 'user3@nav.no'],
                    'failure_count'          => 0,
                    'last_seen_at'           => $this->getTimestamp(1589956403),
                ],
                [
                    'id'                     => 3,
                    'serial'                 => 'serial4',
                    'platform'               => 'rhel',
                    'assigned_owner'         => ['email' => 'user4@nav.no'],
                    'failure_count'          => 1,
                    'last_seen_at'           => $this->getTimestamp(1589956403),
                ],
                [
                    'id'                     => 4,
                    'serial'                 => 'serial5',
                    'platform'               => 'rhel',
                    'assigned_owner'         => ['email' => 'user5@nav.no'],
                    'failure_count'          => 1,
                    'last_seen_at'           => $this->getTimestamp(1589956403),
                ],
            ],
        ]);

        $kolideApiClient
            ->method('getDeviceFailures')
            ->withConsecutive(
                [3],
                [4]
            )
            ->will($this->onConsecutiveCalls(
                [
                    [
                        'resolved_at' => null,
                        'check_id'    => 7,
                        'timestamp'   => '2020-01-01T16:24:06.000Z',
                        'title'       => 'some failing check',
                    ],
                    [
                        'resolved_at' => '2020-01-02T16:24:06.000Z',
                        'check_id'    => 7,
                        'timestamp'   => '2020-01-01T16:24:06.000Z',
                        'title'       => 'some resolved failing check',
                    ],
                    [
                        'resolved_at' => null,
                        'check_id'    => 123123123,
                        'timestamp'   => '2020-01-01T16:24:06.000Z',
                        'title'       => 'some failing check that should be ignored',
                    ],
                    [
                        'resolved_at' => null,
                        'check_id'    => 15804,
                        'timestamp'   => '2020-01-01T16:24:06.000Z',
                        'title'       => 'some failing check that should be ignored',
                    ]
                ],
                [
                    [
                        'resolved_at' => null,
                        'check_id'    => 8,
                        'timestamp'   => $this->getTimestamp(time() - 3600),
                        'title'       => 'some failing check that is within the allowed grace time for the given severity tag',
                    ],
                ]
            ));

        $kolideApiClient
            ->method('getCheck')
            ->withConsecutive(
                [7],
                [15804],
                [8]
            )
            ->will($this->onConsecutiveCalls(
                ['tags' => ['DANGER']],
                ['tags' => ['INFO']],
                ['tags' => ['NOTICE']]
            ));

        $command = new CheckAndUpdateDevices();
        $command->setApiServerClient($apiServerClient);
        $command->setKolideApiClient($kolideApiClient);

        $application = new Application('Device health checker test');
        $application->setAutoExit(false);
        $application->add($command);

        $applicationTester = new ApplicationTester($application);
        $exitCode = $applicationTester->run([
            $command->getName(),
            '--ignore-checks' => [123123123],
        ]);

        $display = explode(PHP_EOL, trim($applicationTester->getDisplay()));

        $this->assertSame(0, $exitCode, 'Expected exit code to be 0');
        $this->assertSame(
            [
                'Fetched devices from Kolide',
                'Could not find matching Kolide device',
                'No failing checks anymore, device is now healthy',
                'Device is no longer healthy because of the following failing check(s): some failing check',
                'Sent updated Nais device configuration to API server',
            ],
            array_map(fn($msg) => json_decode($msg, true)['message'], $display),
            'Unexpected message in logs'
        );
        $this->assertSame(
            [
                null,
                'serial2-not-in-kolide',
                'serial3',
                'serial4',
                null,
            ],
            array_map(fn($msg) => json_decode($msg, true)['serial'] ?? null, $display),
            'Unexpected serial in logs'
        );
        $this->assertSame(
            [
                null,
                'darwin',
                'linux',
                'linux',
                null,
            ],
            array_map(fn($msg) => json_decode($msg, true)['platform'] ?? null, $display),
            'Unexpected platform in logs'
        );
        $this->assertSame(
            [
                null,
                'user2@nav.no',
                'user3@nav.no',
                'user4@nav.no',
                null,
            ],
            array_map(fn($msg) => json_decode($msg, true)['username'] ?? null, $display),
            'Unexpected username in logs'
        );
    }

    private function getTimestamp(int $time) : string {
        return gmdate("Y-m-d\TH:i:s.v\Z", $time);
    }
}