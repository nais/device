#!/usr/bin/env bash
deviceInfo='{"publicKey": "asd123Client+=", "serial": "123", "platform": "linux"}'

http localhost:80/postDeviceInfo/123 <<< "$deviceInfo"
sleep 5
http localhost:80/getBootstrapConfig/123
