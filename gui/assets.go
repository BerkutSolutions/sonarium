package gui

import "embed"

// Assets contains embedded static UI files.
//
//go:embed static/* static/js/*
var Assets embed.FS
