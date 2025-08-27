#!/usr/bin/env bash
#MISE description="Generate strings for systray"
go tool golang.org/x/tools/cmd/stringer -type=GuiEvent ./internal/systray
go tool mvdan.cc/gofumpt -w ./internal/systray/guievent_string.go
