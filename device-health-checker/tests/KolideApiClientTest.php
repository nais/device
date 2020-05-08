<?php declare(strict_types=1);
namespace Nais\Device;

use PHPUnit\Framework\TestCase;
use GuzzleHttp\Client as HttpClient;
use GuzzleHttp\Handler\MockHandler;
use GuzzleHttp\HandlerStack;
use GuzzleHttp\Psr7\Response;
use GuzzleHttp\Middleware;

/**
 * @coversDefaultClass Nais\Device\KolideApiClient
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

        $checks = (new KolideApiClient('secret', 5, $httpClient))->getAllChecks();

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

        $checks = (new KolideApiClient('secret', 5, $httpClient))->getFailingChecks([2, 3]);

        $this->assertCount(1, $checks, 'Expected 1 checks');
        $this->assertSame(1, $checks[0]['id'], 'Incorrect check result');

        $this->assertCount(2, $clientHistory, 'Expected 2 requests');
        $this->assertStringEndsWith('checks?page=0', (string) $clientHistory[0]['request']->getUri());
        $this->assertStringEndsWith('checks?page=1', (string) $clientHistory[1]['request']->getUri());
    }

    /**
     * @covers ::__construct
     * @covers ::getCheckFailures
     * @covers ::getPaginatedResults
     */
    public function testCanGetCheckFailures() : void {
        $clientHistory = [];
        $httpClient = $this->getMockClient(
            [
                new Response(200, [], json_encode(['page' => 0, 'last_page' => 1, 'data' => [['id' => 1], ['id' => 2]]])),
                new Response(200, [], json_encode(['page' => 1, 'last_page' => 1, 'data' => [['id' => 3], ['id' => 4]]])),
            ],
            $clientHistory
        );

        $failures = (new KolideApiClient('secret', 5, $httpClient))->getCheckFailures(1);

        $this->assertCount(4, $failures, 'Expected 4 failures');

        $this->assertCount(2, $clientHistory, 'Expected 2 requests');
        $this->assertStringEndsWith('checks/1/failures?page=0', (string) $clientHistory[0]['request']->getUri());
        $this->assertStringEndsWith('checks/1/failures?page=1', (string) $clientHistory[1]['request']->getUri());
    }

    /**
     * @covers ::getDeviceBySerial
     */
    public function testGetDeviceBySerialReturnsNullWhenDeviceIsNotFound() : void {
        $clientHistory = [];
        $httpClient = $this->getMockClient(
            [new Response(200, [], '{"last_page":0,"page":0,"data":[]}')],
            $clientHistory
        );

        $this->assertNull((new KolideApiClient('secret', 5, $httpClient))->getDeviceBySerial('serial'), 'Expected method to return null');
        $this->assertCount(1, $clientHistory, 'Expected 1 request');
        $this->assertStringEndsWith('devices?search=serial', (string) $clientHistory[0]['request']->getUri());
    }

    /**
     * @covers ::getDeviceBySerial
     */
    public function testCanGetDeviceBySerial() : void {
        $clientHistory = [];
        $httpClient = $this->getMockClient(
            [new Response(200, [], '{"last_page":0,"page":0,"data":[{"id": 123}]}')],
            $clientHistory
        );

        $this->assertSame(['id' => 123], (new KolideApiClient('secret', 5, $httpClient))->getDeviceBySerial('serial'), 'Expected device');
        $this->assertCount(1, $clientHistory, 'Expected 1 request');
        $this->assertStringEndsWith('devices?search=serial', (string) $clientHistory[0]['request']->getUri());
    }
}