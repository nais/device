#!/usr/bin/env bash
bootstrapConfig=' { "deviceIP":"10.255.240.69", "publicKey":"asd123+=", "tunnelEndpoint":"35.35.35.35:51820", "apiServerIP":"10.255.240.1" }'

http localhost:8080/api/v1/deviceinfo
http localhost:8080/api/v1/bootstrapconfig/123 <<< "$bootstrapConfig"
