<?php declare(strict_types=1);
namespace Nais\Device;

use GuzzleHttp\Client as HttpClient;
use GuzzleHttp\Exception\ClientException;

class KolideApiClient {
    /** @var HttpClient */
    private $client;

    public function __construct(string $token, int $timeout = 5, HttpClient $client = null) {
        $this->client = $client ?: new HttpClient([
            'base_uri' => 'https://k2.kolide.com/api/v0/',
            'timeout'  => $timeout,
            'headers'  => [
                'Authorization' => sprintf('Bearer %s', $token),
                'Accept'        => 'application/json',
            ],
        ]);
    }

    private function getPaginatedResults(string $endpoint) : array {
        $page = 0;
        $lastPage = false;
        $entries = [];

        while (!$lastPage) {
            $response = json_decode($this->client->get($endpoint, [
                'query' => [
                    'page' => $page++,
                ],
            ])->getBody()->getContents(), true);

            $lastPage = $response['page'] === $response['last_page'];
            $entries = array_merge($entries, $response['data']);
        }

        return $entries;
    }

    public function getAllDevices() : array {
        return $this->getPaginatedResults('devices');
    }

    public function getAllChecks() : array {
        return $this->getPaginatedResults('checks');
    }

    public function getFailingChecks(array $ignoredChecks = []) : array {
        return array_values(array_filter($this->getAllChecks(), function(array $check) use ($ignoredChecks) : bool {
            return !in_array($check['id'], $ignoredChecks) && 0 !== $check['failing_device_count'];
        }));
    }

    public function getCheckFailures(int $checkId) : array {
        return $this->getPaginatedResults(sprintf('checks/%d/failures', $checkId));
    }

    public function getDeviceFailures(int $deviceId) : array {
        return $this->getPaginatedResults(sprintf('devices/%d/failures', $deviceId));
    }

    public function getCheck(int $checkId) : ?array {
        try {
            return json_decode($this->client->get(sprintf('checks/%d', $checkId))->getBody()->getContents(), true);
        } catch (ClientException $e) {
            return null;
        }
    }
}