<?php declare(strict_types=1);
function fail(string $message) : void {
    echo trim($message) . PHP_EOL;
    exit(1);
}

foreach (['INCOMPLETE_CHECKS', 'SLACK_WEBHOOK'] as $requiredEnvVar) {
    if (empty(getenv($requiredEnvVar))) {
        fail(sprintf('Missing required environment variable: %s', $requiredEnvVar));
    }
}

$incompleteChecks = json_decode(getenv('INCOMPLETE_CHECKS'), true);

if (json_last_error() !== JSON_ERROR_NONE) {
    fail(sprintf('Unable to decode JSON: %s', json_last_error_msg()));
}

$c = curl_init();

curl_setopt($c, CURLOPT_URL, getenv('SLACK_WEBHOOK'));
curl_setopt($c, CURLOPT_HTTPHEADER, ['Content-Type: application/json']);
curl_setopt($c, CURLOPT_POST, 1);
curl_setopt($c, CURLOPT_POSTFIELDS, json_encode([
    'blocks' => [
        [
            'type' => 'section',
            'text' => [
                'type' => 'mrkdwn',
                'text' => 'The following <https://k2.kolide.com/1401/checks/active?tags%5B%5D=untagged|Kolide checks> are missing tags:',
            ],
        ],
        [
            'type' => 'divider',
        ],
        ...array_map(function(array $c) : array {
            return [
                'type' => 'section',
			    'text' => [
                    'type' => 'mrkdwn',
                    'text' => sprintf(
                        "*<https://k2.kolide.com/1401/checks/%d|%s>* (%d failure%s)\n%s\n\n_Compatibility_: %s\n_Topics_: %s",
                        $c['id'],
                        $c['name'],
                        $c['failing_device_count'],
                        1 !== $c['failing_device_count'] ? 's' : '',
                        $c['description'],
                        implode(', ', $c['compatibility']) ?: 'n/a',
                        implode(', ', $c['topics']) ?: 'n/a',
                    ),
                ],
            ];
        }, $incompleteChecks),
    ],
]));

curl_exec($c);