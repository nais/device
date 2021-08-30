# How to rotate naisdevice-apiserver client secret

## Find app registration
### Option A: Direct link
0. [direct link to app (works at time of writing)](https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationMenuBlade/Credentials/appId/6e45010d-2637-4a40-b91d-d4cbb451fb57/isMSAApp/)

### Option B: Navigate GUI
1. Head over to https://portal.azure.com/
2. In the menu (top left), select Azure Active Directory
3. In the menu to the left, select `App registrations`
4. Select the `All applications` tab
5. Search for `naisdevice-apiserver`
6. In the menu to the left, select `Certificates & secrets`


## Add new client secret
1. Below the `Client secrets` heading, click `+ New client secret`
2. Enter description `naisdevice-apiserver` and 24 month expiry
3. Click `Add`
40. Copy the Value shown for the new client secret row in the table


## Update secret used by naisdevice-apiserver
1. Add new secret version
```bash
read -s client_secret && gcloud --project nais-device secrets versions add azure-client-secret --data-file <(echo "$client_secret")
```
2. Restart apiserver
```
gcloud --project nais-device compute ssh --tunnel-through-iap apiserver -- sudo systemctl restart apiserver
```
3. Wait for logs to show and verify no Azure related errors, CTRL+C to stop the `tail` command.

## Verify new secret is working
1. Kill existing session
```
killall naisdevice
killall naisdevice-systray
killall naisdevice-agent
```

2. Start naisdevice, `connect`, wait for :kekw:
