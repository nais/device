#!/usr/bin/env bash
socket_dir=$(mktemp -d)

cloud_sql_proxy -dir="${socket_dir}/cloud_sql_sockets/" -projects nais-device -instances=nais-device:europe-north1:naisdevice-3824b4c7 &
pid=$!
sleep 5
trap 'kill "$pid"; rm -rf "$socket_dir"' EXIT

rm -r data/
mkdir -p data/

psql "sslmode=disable host=${socket_dir}/cloud_sql_sockets/nais-device:europe-north1:naisdevice-3824b4c7 user=apiserver dbname=naisdevice" -o data/devices.csv --csv -c "select * from device;"
psql "sslmode=disable host=${socket_dir}/cloud_sql_sockets/nais-device:europe-north1:naisdevice-3824b4c7 user=apiserver dbname=naisdevice" -o data/gateways.csv --csv -c "select * from gateway;"
psql "sslmode=disable host=${socket_dir}/cloud_sql_sockets/nais-device:europe-north1:naisdevice-3824b4c7 user=apiserver dbname=naisdevice" -o data/sessions.csv --csv -c "select * from session;"
