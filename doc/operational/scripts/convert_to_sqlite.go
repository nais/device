package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func convert(path string, convertFn func(map[string]string)) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("open: %v", err)
		panic(err)
	}

	csvReader := csv.NewReader(file)
	csvHeader, err := csvReader.Read()
	if err != nil {
		fmt.Printf("get csv keys row: %v", err)
		panic(err)
	}

	for {
		line, err := csvReader.Read()
		if err == io.EOF {
			break
		}

		data := make(map[string]string)
		for i, value := range line {
			data[csvHeader[i]] = value
		}

		convertFn(data)
	}
}

func main() {
	// generated with: psql "sslmode=disable host=$HOME/cloud_sql_sockets/nais-device:europe-north1:naisdevice-3824b4c7 user=apiserver dbname=naisdevice" -o devices.csv --csv -c "select * from device"
	convert("data/devices.csv", convertDevice)

	// generated with: psql "sslmode=disable host=$HOME/cloud_sql_sockets/nais-device:europe-north1:naisdevice-3824b4c7 user=apiserver dbname=naisdevice" -o gateways.csv --csv -c "select * from gateway"
	convert("data/gateways.csv", convertGateway)

	// generated with: psql "sslmode=disable host=$HOME/cloud_sql_sockets/nais-device:europe-north1:naisdevice-3824b4c7 user=apiserver dbname=naisdevice" -o sessions.csv --csv -c "select * from session"
	convert("data/sessions.csv", convertSessions)
}

/*
# source csv
id,username,serial,psk,platform,healthy,public_key,ip,last_updated,kolide_last_seen

# target tables
sqlite> PRAGMA table_info(devices);
0|id|INTEGER|0||1
1|username|TEXT|1||0
2|serial|TEXT|1||0
3|platform|TEXT|1||0
4|healthy|BOOLEAN|1|0|0
5|last_updated|TEXT|0||0
6|public_key|TEXT|1||0
7|ip|TEXT|1||0
*/
func convertDevice(device map[string]string) {
	if device["last_updated"] == "" {
		device["last_updated"] = "2006-01-02 15:04:05.999999+00"
	}

	fmt.Printf("INSERT INTO devices(id, username, serial, platform, healthy, last_updated, public_key, ip) VALUES(%s, '%s', '%s', '%s', %s, '%s', '%s', '%s');\n",
		device["id"],
		device["username"],
		device["serial"],
		device["platform"],
		convertBool(device["healthy"]),
		convertTime(device["last_updated"]),
		device["public_key"],
		device["ip"],
	)
}

/*
# source csv
id,name,access_group_ids,endpoint,public_key,ip,routes,requires_privileged_access,password_hash

# target tables
sqlite> PRAGMA table_info(gateways);
0|name|TEXT|0||1
1|endpoint|TEXT|1||0
2|public_key|TEXT|1||0
3|ip|TEXT|1||0
4|requires_privileged_access|BOOLEAN|1|0|0
5|password_hash|TEXT|1||0

sqlite> PRAGMA table_info(gateway_routes);
0|gateway_name|TEXT|1||1
1|route|TEXT|1||2

sqlite> PRAGMA table_info(gateway_access_group_ids);
0|gateway_name|TEXT|1||1
1|group_id|TEXT|1||2
*/
func convertGateway(gateway map[string]string) {
	fmt.Printf("INSERT INTO gateways(name, endpoint, public_key, ip, requires_privileged_access, password_hash) VALUES('%s', '%s', '%s', '%s', %s, '%s');\n",
		gateway["name"],
		gateway["endpoint"],
		gateway["public_key"],
		gateway["ip"],
		convertBool(gateway["requires_privileged_access"]),
		gateway["password_hash"],
	)

	routes := make(map[string]bool)
	for route := range strings.SplitSeq(gateway["routes"], ",") {
		if routes[route] {
			continue
		}
		routes[route] = true
		fmt.Printf("INSERT INTO gateway_routes(gateway_name, route) VALUES('%s', '%s');\n",
			gateway["name"],
			route,
		)
	}

	groupIDs := make(map[string]bool)
	for groupID := range strings.SplitSeq(gateway["access_group_ids"], ",") {
		if groupIDs[groupID] {
			continue
		}
		groupIDs[groupID] = true

		fmt.Printf("INSERT INTO gateway_access_group_ids(gateway_name, group_id) VALUES('%s', '%s');\n",
			gateway["name"],
			groupID,
		)
	}
}

/*
"cat - > backupfile.tar"
# source csv
key,device_id,groups,object_id,expiry

# target tables
sqlite> PRAGMA table_info(sessions);
0|key|TEXT|1||0
1|expiry|TEXT|1||0
2|device_id|INTEGER|1||0
3|object_id|TEXT|1||0

sqlite> PRAGMA table_info(session_access_group_ids);
0|session_key|TEXT|1||1
1|group_id|TEXT|1||2
*/
func convertSessions(sessions map[string]string) {
	fmt.Printf("INSERT INTO sessions(key, expiry, device_id, object_id) VALUES('%s', '%s', %s, '%s');\n",
		sessions["key"],
		convertTime(sessions["expiry"]),
		sessions["device_id"],
		sessions["object_id"],
	)

	for groupID := range strings.SplitSeq(sessions["groups"], ",") {
		fmt.Printf("INSERT INTO session_access_group_ids(session_key, group_id) VALUES('%s', '%s');\n",
			sessions["key"],
			groupID,
		)
	}
}

func convertBool(b string) string {
	switch b {
	case "t":
		return "true"
	default:
		return "false"
	}
}

func convertTime(t string) string {
	format := "2006-01-02 15:04:05.999999+00"
	dt, err := time.Parse(format, t)
	if err != nil {
		panic(err)
	}
	return dt.Format(time.RFC3339Nano)
}
