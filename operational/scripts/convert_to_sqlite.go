package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	// generated with: psql "sslmode=disable host=$(pwd)/nais-device:europe-north1:naisdevice-3824b4c7 user=apiserver dbname=naisdevice" -o devices.csv --csv -c "select * from device"
	devicesFile, err := os.Open("data/devices.csv")
	if err != nil {
		fmt.Printf("open: %v", err)
		os.Exit(1)
	}

	devices := csv.NewReader(devicesFile)
	deviceKeys, err := devices.Read()
	if err != nil {
		fmt.Printf("get csv keys row: %v", err)
		os.Exit(1)
	}

	for {
		device, err := devices.Read()
		if err == io.EOF {
			break
		}

		deviceAsMap := make(map[string]string)
		for i, value := range device {
			deviceAsMap[deviceKeys[i]] = value
		}

		convertDevice(deviceAsMap)
	}

	// generated with: psql "sslmode=disable host=$(pwd)/nais-device:europe-north1:naisdevice-3824b4c7 user=apiserver dbname=naisdevice" -o gateways.csv --csv -c "select * from gateway"
	gatewaysFile, err := os.Open("data/gateways.csv")
	if err != nil {
		fmt.Printf("open: %v", err)
		os.Exit(1)
	}

	gateways := csv.NewReader(gatewaysFile)
	gatewayKeys, err := gateways.Read()
	if err != nil {
		fmt.Printf("get csv keys row: %v", err)
		os.Exit(1)
	}

	for {
		gateway, err := gateways.Read()
		if err == io.EOF {
			break
		}

		gatewayAsMap := make(map[string]string)
		for i, value := range gateway {
			gatewayAsMap[gatewayKeys[i]] = value
		}

		convertGateway(gatewayAsMap)
	}
}

func convertDevice(device map[string]string) {
	// sqlite> PRAGMA table_info(devices);
	// 0|id|INTEGER|0||1
	// 1|username|TEXT|1||0
	// 2|serial|TEXT|1||0
	// 3|platform|TEXT|1||0
	// 4|healthy|BOOLEAN|1|0|0
	// 5|last_updated|TEXT|0||0
	// 6|public_key|TEXT|1||0
	// 7|ip|TEXT|1||0

	fmt.Printf("INSERT INTO devices(id, username, serial, platform, healthy, last_updated, public_key, ip) VALUES(%s, '%s', '%s', '%s', %s, '%s', '%s', '%s');\n",
		device["id"],
		device["username"],
		device["serial"],
		device["platform"],
		convertBool(device["healthy"]),
		device["last_updated"],
		device["public_key"],
		device["ip"],
	)
}

func convertGateway(gateway map[string]string) {
	// sqlite> PRAGMA table_info(gateways);
	// 0|name|TEXT|0||1
	// 1|endpoint|TEXT|1||0
	// 2|public_key|TEXT|1||0
	// 3|ip|TEXT|1||0
	// 4|requires_privileged_access|BOOLEAN|1|0|0
	// 5|password_hash|TEXT|1||0

	// sqlite> PRAGMA table_info(gateway_routes);
	// 0|gateway_name|TEXT|1||1
	// 1|route|TEXT|1||2

	// sqlite> PRAGMA table_info(gateway_access_group_ids);
	// 0|gateway_name|TEXT|1||1
	// 1|group_id|TEXT|1||2

	// id,name,access_group_ids,endpoint,public_key,ip,routes,requires_privileged_access,password_hash

	fmt.Printf("INSERT INTO gateways(name, endpoint, public_key, ip, requires_privileged_access, password_hash) VALUES('%s', '%s', '%s', '%s', %s, '%s');\n",
		gateway["name"],
		gateway["endpoint"],
		gateway["public_key"],
		gateway["ip"],
		convertBool(gateway["requires_privileged_access"]),
		gateway["password_hash"],
	)

	routes := make(map[string]bool)
	for _, route := range strings.Split(gateway["routes"], ",") {
		if routes[route] == true {
			continue
		}
		routes[route] = true
		fmt.Printf("INSERT INTO gateway_routes(gateway_name, route) VALUES('%s', '%s');\n",
			gateway["name"],
			route,
		)
	}

	groupIDs := make(map[string]bool)
	for _, groupID := range strings.Split(gateway["access_group_ids"], ",") {
		if groupIDs[groupID] == true {
			continue
		}
		groupIDs[groupID] = true

		fmt.Printf("INSERT INTO gateway_access_group_ids(gateway_name, group_id) VALUES('%s', '%s');\n",
			gateway["name"],
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
