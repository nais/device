#!/usr/bin/env bash
wait="bash -c 'read -p \"Press enter to continue\"'"
if [[ -z "$(command -v tmux)" ]]; then
	echo "tmux is required to run this script"
	exit 1
fi

window_name="naisdevice-$(date +"%H%M%S")"

enrollments_port=8081
enrollments_url="http://localhost:${enrollments_port}/enrollments"

env=(
	"ENROLLER_LOCALLISTENADDR=\":${enrollments_port}\""
	"APISERVER_WIREGUARDIP=\"127.0.0.1/24\""
	"APISERVER_WIREGUARDIPV6=\"::1/64\""
	"APISERVER_AUTOENROLLENABLED=\"true\""
	"APISERVER_AUTOENROLLMENTSURL=\"${enrollments_url}\""
	"APISERVER_DEVICEAUTHENTICATIONPROVIDER=\"azure\""
	"APISERVER_LOGLEVEL=\"debug\""
)

# shellcheck disable=SC2116
env_joined="$(IFS=' ' echo "${env[*]}")"

session_id="$window_name"
if [ -z "$TMUX" ]; then
	tmux new-session -n "$window_name" -s "$session_id" -d
else
	session_id="$(tmux new-window -n "$window_name" -P | cut -d ':' -f 1)"
fi
window_id="${session_id}:${window_name}"

# make sure we have enough panes

declare -A panes=(
	[apiserver]="go run ./cmd/apiserver"
	[enroller]="go run ./cmd/enroller"
	[deviceagent]="go run ./cmd/naisdevice-agent --no-helper --custom-enroll-url 'http://localhost:8080/enroll'"
)

for pane in "${!panes[@]}"; do
	cmd="${panes[$pane]}"
	tmux split-window -t "${window_id}" -v "$env_joined $cmd; $wait"
	tmux select-pane -T "$cmd"
done

tmux select-layout -t "$window_id" even-vertical

tmux set -w -t "$window_id" pane-border-status top
tmux set -w -t "$window_id" pane-border-format "#{pane_index} #{pane_title}"

if [ -z "$TMUX" ]; then
	# only attach if we are not already in a tmux session
	tmux attach -t "$session_id"
fi
