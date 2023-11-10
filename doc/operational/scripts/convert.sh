#!/usr/bin/env bash
set -e
socket_dir=$(mktemp -d)
connection_name="nais-io:europe-north1:naisdevice-e7bdb702"
project="$(cut -d ':' -f 1 <<< "$connection_name")"

cloud_sql_proxy "${connection_name}?unix-socket-path=${socket_dir}/${connection_name}" &
export PGPASSWORD=$(gcloud --project "$project" compute ssh --tunnel-through-iap naisdevice-apiserver -- sudo grep "DBCONNDSN" /etc/default/apiserver | cut -d ' ' -f 4 | cut -d '=' -f 2)
pid=$!
sleep 2

trap 'kill "$pid"; rm -rf "$socket_dir"' EXIT

rm -f new.db
rm -r data/
mkdir -p data/

psql "sslmode=disable host=${socket_dir}/${connection_name} user=apiserver dbname=naisdevice" -o data/devices.csv --csv -c "select * from device;"
psql "sslmode=disable host=${socket_dir}/${connection_name} user=apiserver dbname=naisdevice" -o data/gateways.csv --csv -c "select * from gateway;"
psql "sslmode=disable host=${socket_dir}/${connection_name} user=apiserver dbname=naisdevice" -o data/sessions.csv --csv -c "select * from session;"

go run ./convert_to_sqlite.go > data/inserts.sql
sqlite3 new.db < ../../pkg/apiserver/database/schema/0001_schema.up.sql
sqlite3 new.db < data/inserts.sql
sqlite3 new.db <<< "CREATE TABLE schema_migrations (version uint64, dirty bool); INSERT INTO schema_migrations VALUES (1, false);"

gcloud --project "$project" compute scp --tunnel-through-iap new.db naisdevice-apiserver:.

cat <<EOF | gcloud --project "$project" compute ssh --zone=europe-north1-a --tunnel-through-iap naisdevice-apiserver
sudo mkdir /var/lib/naisdevice/
sudo install -o root -g root -m 640 ./new.db /var/lib/naisdevice/apiserver.db
sudo grep -q APISERVER_DBPATH /etc/default/apiserver || sudo tee -a /etc/default/apiserver <<< 'APISERVER_DBPATH="/var/lib/naisdevice/apiserver.db"'
sudo apt-get update
sudo apt-get install apiserver
EOF
