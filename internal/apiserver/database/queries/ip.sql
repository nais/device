-- name: GetAllocatedIPv6 :many
SELECT ipv6 FROM devices WHERE ipv6 != ''
UNION ALL
SELECT ipv6 FROM gateways WHERE ipv6 != '';
