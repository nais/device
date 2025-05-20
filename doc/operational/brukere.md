# Håndtering av brukere

## Bruker har fått endret e-post

Av og til bytter folk navn, og da får de ny e-post, men de er fortsatt samme person.
Da må vi inn i tenants sin Apiserver og oppdatere `devices`-radene som har den gamle e-posten.
Hvis man velger å slette brukeren fra Apiserveren må de huske å slette sin lokale Naisdevice config: 

```shell
# Mac
rm -r "~/Library/Application Support/naisdevice/"
```

I stedet for å slette brukere kan man også oppdatere databasen.

Guiden nedenfor bruker Nav som eksempel.

1. Start med å SSH inn på riktig Apiserver

   ```shell
   gcloud compute ssh --zone "europe-north1-a" "apiserver" --project "nais-device" --tunnel-through-iap
   ```
3. Koble deg på SQLite-databasen

   ```shell
   sudo sqlite3 /var/lib/naisdevice/apiserver.db
   ```
4. List brukere med utdatert e-post

   ```sql
   select * from devices where username = 'forrige.epost@nav.no';
   ```
5. Oppdater radene med ny e-post

   ```sql
   update devices set username = 'ny.epost@nav.no' where username = 'forrige.epost@nav.no' limit 1;
   ```

PS: E-post bruker ofte stor forbokstav. `ny.epost@nav.no` er ikke den samme som `Ny.Epost@nav.no`, og brukere logger inn med forskjellig bruk av store bokstaver.
