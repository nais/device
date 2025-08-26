#!/usr/bin/env bash
if [[ -z "$(command -v tmux)" ]]; then
	echo "tmux is required to run this script"
	exit 1
fi

window_name="naisdevice-$(date +"%H%M%S")"

export APISERVER_WIREGUARDIP="127.0.0.1/24"
export APISERVER_WIREGUARDIPV6="::1/64"
export APISERVER_AUTOENROLLENABLED="true"
export APISERVER_AUTOENROLLMENTSURL="http://localhost:8081/enrollments"
export APISERVER_DEVICEAUTHENTICATIONPROVIDER="azure" # can be set to mock, azure or google

export ENROLLER_LOCALLISTENADDR=":8081"

session_id="$window_name"
if [ -z "$TMUX" ]; then
	tmux new-session -n "$window_name" -s "$session_id" -d
else
	tmux new-window -n "$window_name" -P
	session_id=$(tmux display-message -p '#S' 2>/dev/null)
fi
window_id="${session_id}:${window_name}"

# make sure we have enough panes

declare -A panes=(
	[deviceagent]="go run ./cmd/apiserver"
	[enroller]="go run ./cmd/enroller"
	[apiserver]="go run ./cmd/naisdevice-agent --custom-enroll-url \"$APISERVER_AUTOENROLLMENTSURL\""
)

for pane in "${!panes[@]}"; do
	cmd="${panes[$pane]}"
	tmux split-window -t "${window_id}" -v "$cmd"
	tmux select-pane -T "$cmd"
done
tmux select-layout -t "$window_id" even-vertical

tmux set -w -t "$window_id" pane-border-status top
tmux set -w -t "$window_id" pane-border-format "#{pane_index} #{pane_title}"
tmux attach -t "$session_id"
