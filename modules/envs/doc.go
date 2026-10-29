// Package envs loads 1Password environment variables with parallel fetches
// and caches them encrypted locally for instant shell startup.
//
// By default it loads items matching "envs-*" from the Private vault.
// Configure via ~/.config/cly/config.yaml (or config.local.yaml):
//
//	envs:
//	  account: "my.1password.com"
//	  vault: "Private"
//	  sections: true
//	  cache: true
//	  store: ""        # "keychain", "age", or empty for OS default
//	  secrets: ["envs-*"]
//
// Cache backends:
//
//   - macOS: Keychain entry "nsx-envs/__blob__" via the security CLI,
//     readable by caches written by the original nsx implementation.
//   - Linux (or store: age anywhere): an age-encrypted file at
//     ~/.local/state/cly/envs.age, decryptable with the age identity at
//     ~/.config/cly/envs.age.key or any unencrypted SSH private key in
//     ~/.ssh (ed25519/RSA). The blob format matches the Keychain entry.
//
// Command overview:
//
//	cly envs                 Fetch from 1Password → refresh the cache
//	cly envs --fish          Read from cache → fish format (instant)
//	cly envs --plain         Read from cache → KEY=VALUE format (instant)
//	cly envs --refresh       Force re-fetch from 1Password
//	cly envs install fish    Write shell startup integration
package envs
