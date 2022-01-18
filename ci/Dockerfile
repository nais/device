FROM ubuntu

LABEL org.opencontainers.image.source=https://github.com/nais/device

ARG FPM_VERSION=1.13.1
ARG GCLOUD_VERSION=368.0.0
ARG GO_VERSION=1.17.6

WORKDIR /root

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update -qq
RUN apt-get install -qq --yes build-essential libgtk-3-dev libappindicator3-dev ruby ruby-dev rubygems jq curl imagemagick


RUN gem install --no-document fpm -v "$FPM_VERSION"

RUN curl -L "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
    | tar -xzC /usr/local

RUN curl -L "https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-${GCLOUD_VERSION}-linux-x86_64.tar.gz" \
  | tar -xzC /root

ENV PATH $PATH:/root/google-cloud-sdk/bin:/usr/local/go/bin

RUN gcloud components install beta --quiet \
    && rm -rf $(find google-cloud-sdk/ -regex ".*/__pycache__") \
    && rm -rf google-cloud-sdk/.install/.backup


