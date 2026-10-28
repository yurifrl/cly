// Package envs loads 1Password environment variables with parallel fetches
// and caches them in the macOS Keychain for instant shell startup.
//
// By default it loads items matching "envs-*" from the Private vault.
// Configure via ~/.config/cly/config.yaml (or config.local.yaml):
//
//	envs:
//	  account: "my.1password.com"
//	  vault: "Private"
//	  sections: true
//	  keychain: true
//	  secrets: ["envs-*"]
//
// Command overview:
//
//	cly envs                 Fetch from 1Password → store in Keychain
//	cly envs --fish          Read from Keychain → fish format (instant)
//	cly envs --plain         Read from Keychain → KEY=VALUE format (instant)
//	cly envs --refresh       Force re-fetch from 1Password
//	cly envs install fish    Write shell startup integration
//
// The Keychain entry reuses the "nsx-envs" service name so caches written
// by the original nsx implementation stay valid.
package envs
