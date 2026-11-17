//go:build ignore

// Template overlay file used by Wails v3 during Android c-shared builds; excluded from host gopls analysis.
package main

import "github.com/wailsapp/wails/v3/pkg/application"

func init() {
	// Register main function to be called when the Android app initializes
	// This is necessary because in c-shared build mode, main() is not automatically called
	application.RegisterAndroidMain(main)
}
