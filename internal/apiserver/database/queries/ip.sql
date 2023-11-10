-- name: GetLastUsedIPV6 :one
SELECT CAST(MAX(MAX(devices.ipv6), MAX(gateways.ipv6)) AS text) AS ipv6
FROM devices, gateways;
