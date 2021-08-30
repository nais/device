# How to rotate naisdevice-apiserver client secret

## Find app registration
### Option A: Direct link
[direct link to app (works at time of writing)](https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationMenuBlade/Credentials/appId/6e45010d-2637-4a40-b91d-d4cbb451fb57/isMSAApp/)

### Option B: Navigate GUI
1. Head over to https://portal.azure.com/
1. In the menu (top left), select Azure Active Directory
1. In the menu to the left, select `App registrations`
1. Select the `All applications` tab
1. Search for `naisdevice-apiserver`
1. In the menu to the left, select `Certificates & secrets`


## Add new client secret
1. Below the `Client secrets` heading, click `+ New client secret`
1. Enter description `naisdevice-apiserver` and 24 month expiry
1. Click `Add`
1. Copy the Value shown for the new client secret row in the table
1. Add new secret version
   ```
   read -s client_secret
   ```
1. Add secret to Google secret manager
   ```
   gcloud --project nais-device secrets versions add azure-client-secret --data-file <(echo "$client_secret"); unset client_secret
   ```
1. Restart apiserver
   ```
   gcloud --project nais-device compute ssh --tunnel-through-iap apiserver -- "sudo systemctl restart apiserver && tail -f -n 0 /var/log/naisdevice/apiserver.json"
   ```
1. Wait for logs to show and verify no Azure related errors, CTRL+C to stop the `tail` command.

## Verify new secret is working
1. Kill existing session
   ```
   killall naisdevice
   killall naisdevice-systray
   killall naisdevice-agent
   ```

1. Start naisdevice, `connect`, wait for :kekw:
