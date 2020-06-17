#!/bin/sh
user=$( echo "show State:/Users/ConsoleUser" | scutil | awk '/Name :/ && ! /loginwindow/ { print $3 }' )

daemon_name="io.naisdevice.helper"
destination="/Library/LaunchDaemons/${daemon_name}.plist"

launchctl list | grep -q "$daemon_name" && launchctl unload "$destination"

config_dir="/Users/${user}/Library/Application Support/naisdevice"

cat << EOF > "$destination"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>KeepAlive</key>
        <true/>
        <key>Label</key>
        <string>io.naisdevice.helper</string>
        <key>Nice</key>
        <integer>0</integer>
        <key>Program</key>
        <string>/opt/naisdevice/bin/device-agent-helper</string>
        <key>ProgramArguments</key>
        <array>
                <string>/opt/naisdevice/bin/device-agent-helper</string>
                <string>--interface</string>
                <string>utun69</string>
                <string>--bootstrap-config</string>
                <string>$config_dir/bootstrapconfig.json</string>
                <string>--pid-file</string>
                <string>$config_dir/pid</string>
        </array>
        <key>RunAtLoad</key>
        <true/>
        <key>StandardErrorPath</key>
        <string>/Library/Logs/device-agent-helper-err.log</string>
        <key>StandardOutPath</key>
        <string>/Library/Logs/device-agent-helper-out.log</string>
</dict>
</plist>

EOF

launchctl load "$destination"

chmod 600 "$destination"

echo "Installed service $daemon_name"
