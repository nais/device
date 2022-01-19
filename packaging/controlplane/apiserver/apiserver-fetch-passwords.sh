#!/usr/bin/env bash
apikeys=""
for secret in $(gcloud secrets list --filter "labels.type:api-server-password" --format "value(name)");do
        gateway_name=$(echo "$secret" | cut -d'_' -f2)
        apikey=$(gcloud secrets versions access latest --secret $secret)
        apikeys+="$gateway_name:$apikey,"
done

apikey=$(gcloud secrets versions access latest --secret apiserver-admin-password)
apikeys+="admin:$apikey"

echo -n ${apikeys::-1}

