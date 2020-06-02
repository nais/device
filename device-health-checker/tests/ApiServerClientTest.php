<?php declare(strict_types=1);
namespace Nais\Device;

use PHPUnit\Framework\TestCase;
use GuzzleHttp\Client as HttpClient;
use GuzzleHttp\Handler\MockHandler;
use GuzzleHttp\HandlerStack;
use GuzzleHttp\Psr7\Response;
use GuzzleHttp\Middleware;

/**
 * @coversDefaultClass Nais\Device\ApiServerClient
 */
class ApiServerClientTest extends TestCase {
    private function getMockClient(array $responses, array &$history = []) : HttpClient {
        $handler = HandlerStack::create(new MockHandler($responses));
        $handler->push(Middleware::history($history));

        return new HttpClient(['handler' => $handler]);
    }

    /**
     * @covers ::__construct
     * @covers ::getDevices
     */
    public function testCanGetAllDevices() : void {
        $clientHistory = [];
        $httpClient = $this->getMockClient(
            [new Response(
                200, [],
                '[{"serial":"serial","psk":"psk","lastUpdated":null,"isHealthy":true,"publicKey":"pubkey","ip":"1.2.3.4","username":"user@nav.no"}]'
            )],
            $clientHistory
        );

        $devices = (new ApiServerClient('username', 'password', 5, $httpClient))->getDevices();

        $this->assertCount(1, $devices, 'Expected 1 device');
        $this->assertCount(1, $clientHistory, 'Expected 1 request');
        $this->assertStringEndsWith('/devices', (string) $clientHistory[0]['request']->getUri(), 'Incorrect request');
    }

    /**
     * @covers ::updateDevices
     */
    public function testCanUpdateDevices() : void {
        $clientHistory = [];
        $httpClient = $this->getMockClient(
            [new Response(200)],
            $clientHistory
        );

        (new ApiServerClient('username', 'password', 5, $httpClient))->updateDevices([
            [
                'serial' => 'serial1',
                'platform' => 'darwin',
                'username' => 'user1@nav.no',
                'isHealthy' => true,
            ],
            [
                'serial' => 'serial2',
                'platform' => 'darwin',
                'username' => 'user2@nav.no',
                'isHealthy' => false,
            ],
        ]);

        $this->assertCount(1, $clientHistory, 'Expected 1 request');
        $this->assertSame('PUT', $clientHistory[0]['request']->getMethod(), 'Expected HTTP PUT');
        $this->assertSame(
            '[{"serial":"serial1","platform":"darwin","username":"user1@nav.no","isHealthy":true},{"serial":"serial2","platform":"darwin","username":"user2@nav.no","isHealthy":false}]',
            $clientHistory[0]['request']->getBody()->getContents(),
            'Unexpected request body'
        );
    }
}