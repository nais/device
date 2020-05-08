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
                '[{"serial":"serial","psk":"psk","lastCheck":null,"isHealthy":true,"publicKey":"pubkey","ip":"1.2.3.4","username":"user@nav.no"}]'
            )],
            $clientHistory
        );

        $devices = (new ApiServerClient('username', 'password', 5, $httpClient))->getDevices();

        $this->assertCount(1, $devices, 'Expected 1 device');
        $this->assertCount(1, $clientHistory, 'Expected 1 request');
        $this->assertStringEndsWith('/devices', (string) $clientHistory[0]['request']->getUri(), 'Incorrect request');
    }
}