<?php declare(strict_types=1);
namespace Nais\Device;

use GuzzleHttp\Client as HttpClient;

class ApiServerClient {
    /** @var HttpClient */
    private $client;

    /**
     * Class constructor
     *
     * @param string $username HTTP Basic Auth username
     * @param string $password HTTP Basic Auth password
     * @param int $timeout Request timeout
     * @param HttpClient $client Pre-configured Guzzle client
     */
    public function __construct(string $username, string $password, int $timeout = 5, HttpClient $client = null) {
        $schema = 'http';
        $host   = $_SERVER['APISERVER_HOST'] ?? '10.255.240.1';
        $port   = $_SERVER['APISERVER_PORT'] ?? '';

        if (443 === (int) $port) {
            $schema = 'https';
            $port   = '';
        }

        $this->client = $client ?: new HttpClient([
            'base_uri' => trim(sprintf('%s://%s:%s', $schema, $host, $port), ':'),
            'timeout'  => $timeout,
            'auth'     => [$username, $password],
        ]);
    }

    /**
     * Get all devices
     *
     * @return array
     */
    public function getDevices() : array {
        return json_decode($this->client->get('/devices')->getBody()->getContents(), true) ?: [];
    }

    /**
     * Update devices
     *
     * @param array $devices List of devices to update
     * @return void
     */
    public function updateDevices(array $devices) : void {
        $this->client->put('/devices/health', ['json' => $devices]);
    }
}