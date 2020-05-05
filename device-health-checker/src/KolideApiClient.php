<?php declare(strict_types=1);
namespace Nais;

use GuzzleHttp\Client as HttpClient;

class KolideApiClient {
    private $client;

    public function __construct(string $token, HttpClient $client = null) {
        $this->client = $client ?: new HttpClient([
            'base_uri' => 'https://k2.kolide.com/api/v0/',
            'timeout'  => 3,
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

    public function getAllChecks() : array {
        return $this->getPaginatedResults('checks');
    }

    public function getFailingChecks(array $ignoredChecks = []) : array {
        return array_filter($this->getAllChecks(), function(array $check) use ($ignoredChecks) : bool {
            return !in_array($check['id'], $ignoredChecks) && 0 !== $check['failing_device_count'];
        });
    }

    public function getCheckFailures(int $checkId) : array {
        return $this->getPaginatedResults(sprintf('checks/%d/failures', $checkId));
    }
}