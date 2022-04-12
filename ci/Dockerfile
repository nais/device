FROM ubuntu

LABEL org.opencontainers.image.source=https://github.com/nais/device

ARG FPM_VERSION=1.13.1
ARG GCLOUD_VERSION=368.0.0
ARG GO_VERSION=1.18
WORKDIR /root

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update -qq
RUN apt-get install -qq --yes build-essential libgtk-3-dev libappindicator3-dev ruby ruby-dev rubygems jq curl imagemagick lsb-release


RUN gem install --no-document fpm -v "$FPM_VERSION"

RUN curl -L "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
    | tar -xzC /usr/local

RUN curl -L "https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-${GCLOUD_VERSION}-linux-x86_64.tar.gz" \
  | tar -xzC /root

RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg \
    && echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
      $(lsb_release -cs) stable" > /etc/apt/sources.list.d/docker.list \
    && apt-get update -qq \
    && apt-get install -qq --yes docker-ce docker-ce-cli containerd.io

ENV PATH $PATH:/root/google-cloud-sdk/bin:/usr/local/go/bin

RUN gcloud components install beta --quiet \
    && rm -rf $(find google-cloud-sdk/ -regex ".*/__pycache__") \
    && rm -rf google-cloud-sdk/.install/.backup
