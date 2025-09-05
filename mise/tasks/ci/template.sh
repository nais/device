#!/usr/bin/env bash

env "$(xargs -0 <release_asset_vars.env) " \
	envsubst "$(sed 's/^/$/' release_asset_vars.env)"
