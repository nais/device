# install prereqs
sudo apt-get install ca-certificates curl apt-transport-https gnupg

# add repo signing key
```
HTTPS_PROXY=webproxy-internett.nav.no:8088 curl -L https://europe-north1-apt.pkg.dev/doc/repo-signing-key.gpg | \
  gpg --dearmor | sudo tee /etc/apt/trusted.gpg.d/nais-ppa-google-artifact-registry.gpg
```

# add repository
```
echo 'deb [arch=amd64] https://europe-north1-apt.pkg.dev/projects/naisdevice controlplane main' | \
  sudo tee europe_north1_apt_pkg_dev_projects_naisdevice.list
```

# update repo
```
sudo apt update
sudo apt install gateway-agent
```
