<?php declare(strict_types=1);
namespace Nais;

use PHPUnit\Framework\TestCase;
use GuzzleHttp\Client as HttpClient;
use GuzzleHttp\Handler\MockHandler;
use GuzzleHttp\HandlerStack;
use GuzzleHttp\Psr7\Response;
use GuzzleHttp\Middleware;

/**
 * @coversDefaultClass Nais\KolideApiClient
 */
class KolideApiClientTest extends TestCase {
    private function getMockClient(array $responses, array &$history = []) : HttpClient {
        $handler = HandlerStack::create(new MockHandler($responses));
        $handler->push(Middleware::history($history));

        return new HttpClient(['handler' => $handler]);
    }

    /**
     * @covers ::__construct
     * @covers ::getAllChecks
     * @covers ::getPaginatedResults
     */
    public function testCanGetAllChecks() : void {
        $clientHistory = [];
        $httpClient = $this->getMockClient(
            [
                new Response(200, [], json_encode(['page' => 0, 'last_page' => 1, 'data' => [['id' => 1], ['id' => 2]]])),
                new Response(200, [], json_encode(['page' => 1, 'last_page' => 1, 'data' => [['id' => 3], ['id' => 4]]])),
            ],
            $clientHistory
        );

        $checks = (new KolideApiClient('secret', $httpClient))->getAllChecks();

        $this->assertCount(4, $checks, 'Expected 4 checks');

        $this->assertCount(2, $clientHistory, 'Expected 2 requests');
        $this->assertStringEndsWith('checks?page=0', (string) $clientHistory[0]['request']->getUri());
        $this->assertStringEndsWith('checks?page=1', (string) $clientHistory[1]['request']->getUri());
    }

    /**
     * @covers ::__construct
     * @covers ::getFailingChecks
     */
    public function testCanGetFailingChecks() : void {
        $clientHistory = [];
        $httpClient = $this->getMockClient(
            [
                new Response(200, [], json_encode(['page' => 0, 'last_page' => 1, 'data' => [['id' => 1, 'failing_device_count' => 1], ['id' => 2, 'failing_device_count' => 3]]])),
                new Response(200, [], json_encode(['page' => 1, 'last_page' => 1, 'data' => [['id' => 3, 'failing_device_count' => 2], ['id' => 4, 'failing_device_count' => 0]]])),
            ],
            $clientHistory
        );

        $checks = (new KolideApiClient('secret', $httpClient))->getFailingChecks([2, 3]);

        $this->assertCount(1, $checks, 'Expected 1 checks');
        $this->assertSame(1, $checks[0]['id'], 'Incorrect check result');

        $this->assertCount(2, $clientHistory, 'Expected 2 requests');
        $this->assertStringEndsWith('checks?page=0', (string) $clientHistory[0]['request']->getUri());
        $this->assertStringEndsWith('checks?page=1', (string) $clientHistory[1]['request']->getUri());
    }

    /**
     * @covers ::__construct
     * @covers ::getFailingDevices
     */
    public function testCanGetFailingDevices() : void {
        $clientHistory = [];
        $httpClient = $this->getMockClient(
            [
                new Response(200, [], json_encode(['page' => 0, 'last_page' => 1, 'data' => [['id' => 1, 'failing_device_count' => 1], ['id' => 2, 'failing_device_count' => 3]]])),
                new Response(200, [], json_encode(['page' => 1, 'last_page' => 1, 'data' => [['id' => 3, 'failing_device_count' => 2], ['id' => 4, 'failing_device_count' => 0]]])),
                new Response(200, [], json_encode(['page' => 0, 'last_page' => 1, 'data' => [['id' => 1], ['id' => 3]]])),
                new Response(200, [], json_encode(['page' => 1, 'last_page' => 1, 'data' => [['id' => 1], ['id' => 2]]])),
            ],
            $clientHistory
        );

        $devices = (new KolideApiClient('secret', $httpClient))->getFailingDevices([2, 3]);

        $this->assertCount(3, $devices, 'Expected 3 devices');
        $this->assertSame(['id' => 1], $devices[0]);
        $this->assertSame(['id' => 3], $devices[1]);
        $this->assertSame(['id' => 2], $devices[2]);

        $this->assertCount(4, $clientHistory, 'Expected 4 requests');
        $this->assertStringEndsWith('checks?page=0', (string) $clientHistory[0]['request']->getUri());
        $this->assertStringEndsWith('checks?page=1', (string) $clientHistory[1]['request']->getUri());
        $this->assertStringEndsWith('checks/1/failing_devices?page=0', (string) $clientHistory[2]['request']->getUri());
        $this->assertStringEndsWith('checks/1/failing_devices?page=1', (string) $clientHistory[3]['request']->getUri());
    }
}