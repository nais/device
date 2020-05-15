#!/usr/bin/env bash
bootstrapConfig=' { "deviceIP":"10.255.240.69", "publicKey":"asd123+=", "tunnelEndpoint":"35.35.35.35:51820", "apiServerIP":"10.255.240.1" }'

http localhost:80/getDeviceInfo/123
http localhost:80/postBootstrapConfig/123 <<< "$bootstrapConfig"
