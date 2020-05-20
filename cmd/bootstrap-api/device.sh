#!/usr/bin/env bash
deviceInfo='{"publicKey": "asd123Client+=", "serial": "123", "platform": "linux"}'

http localhost:8080/api/v1/deviceinfo <<< "$deviceInfo"
sleep 5
http localhost:8080/api/v1/bootstrapconfig/123
