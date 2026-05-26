# Notifier Codesign Setup

How to set up the Apple Developer certificate so `pkg/notify/swift/build.sh`
can codesign `cly-notifier.app`. Without this, the bundle is ad-hoc signed
and macOS will deny notification permission (notifications fall back to
`beeep`, which has no action buttons).

## Prerequisites

- macOS with Xcode installed
- Apple Developer account (free is fine for personal use — gives you
  "Apple Development" certs valid on your own machine)
- 1Password CLI (`op`) signed in to `my.1password.com`

## One-time setup

### 1. Generate the certificate via Xcode

Xcode handles CSR + cert request + private key pairing automatically.

```bash
open -a Xcode
```

Then:
- **Xcode menu → Settings...** (⌘,)
- **Accounts** tab → select your Apple ID
- Click **Manage Certificates...**
- Click `+` at bottom-left → **Apple Development**
- Close the dialogs

This installs the Apple Development cert + matching private key into your
login keychain.

### 2. Install the WWDR intermediate CA

Apple's developer certs are signed by an intermediate CA that may not be
trusted by default on a fresh Mac. Without it, the cert shows
`CSSMERR_TP_NOT_TRUSTED` and `find-identity -v` returns 0 valid identities.

```bash
curl -fsSL https://www.apple.com/certificateauthority/AppleWWDRCAG3.cer \
  -o /tmp/AppleWWDRCAG3.cer
security add-certificates /tmp/AppleWWDRCAG3.cer
```

(`add-certificates` writes to your login keychain; no sudo needed. The
older `add-trusted-cert -k /Library/Keychains/System.keychain` requires
sudo and isn't needed for codesigning.)

### 3. Verify the identity is usable

```bash
security find-identity -v -p codesigning
```

Should print **exactly one valid identity**, e.g.:
```
1) AF89DDA6E0A4F85849C04B61094DE03D5974410C "Apple Development: Yuri Freire Lima (RM7WNGWNUK)"
   1 valid identities found
```

If it still says **0 valid identities**, see Troubleshooting below.

### 4. Store the identity in 1Password

The `cly-notifier.app` build script reads `CLY_NOTIFIER_SIGN_ID` from
`.env`, which `task envs:op` resolves from `.env.op` → 1Password.

In your **`cly` Secure Note** (`Private` vault, personal account), add a
field named `APPLE_DEVELOPER_ID` with the **full quoted name** from
`find-identity`:

```
Apple Development: Yuri Freire Lima (RM7WNGWNUK)
```

NOT just the Team ID `RM7WNGWNUK` — codesign needs the full identity name
(or the SHA-1 hash) to find the matching private key.

### 5. Build & install

```bash
cly u
```

This:
- Refreshes `.env` from `.env.op` via `op inject`
- Detects the bundle is stale, runs `pkg/notify/swift/build.sh` with
  `CLY_NOTIFIER_SIGN_ID` from `.env` → real Developer ID signature
- Embeds, builds cly, installs to `~/.local/bin/cly`

Then on first `cly every --notify` run, macOS shows the notification
permission prompt for `dev.yurifrl.cly`. Approve once.

## Troubleshooting

### `0 valid identities found`

```bash
security find-identity -v          # all-policy view
security find-identity              # without "valid only" filter
```

If you see your cert but `(CSSMERR_TP_NOT_TRUSTED)` → install WWDR cert
(step 2).

If you see your cert but no private key → the `.cer` was installed without
the matching `.p12`. The cert alone is useless. Re-run step 1 (Xcode
regenerates a fresh CSR locally and pairs the key).

If you see nothing → no cert installed. Run step 1.

### `RM7WNGWNUK: no identity found` during build

The 1Password field has just the Team ID. Update it to the full quoted
name from `find-identity -v -p codesigning` (step 4).

### `errSecInternalComponent` when codesigning

Your login keychain is locked or codesign can't access the private key.

```bash
security unlock-keychain ~/Library/Keychains/login.keychain-db
```

Or run `cly u` from a Terminal session that already has Keychain access
granted.

### Notifications still don't appear after install

```bash
# Check macOS notification permissions:
open "x-apple.systempreferences:com.apple.preference.notifications"
```

Find `cly` (or `dev.yurifrl.cly`). Set to "Alerts", enable
banners, sounds.

If it's not in the list, the daemon hasn't been launched yet — run any
`cly every --notify` command once to trigger registration.

### `CLY_NOTIFIER_DEBUG=1` for verbose tracing

```bash
CLY_NOTIFIER_DEBUG=1 cly every --notify ... -- ...
```

Prints bundle install path, daemon spawn, socket dial, ready handshake,
and any backend selection fallbacks to stderr.

## Backup the signing identity (optional)

If you want to use this identity on another Mac without re-running step 1:

1. **Keychain Access** → find "Apple Development: ..."
2. Right-click → **Export Items...** → save as `.p12` with a strong password
3. Attach the `.p12` to your `cly` 1Password note as a file (and store the
   password as a separate field). Cert public half + private key + password
   are all you need to recreate the identity on a new Mac.
4. On the new Mac: download the `.p12`, double-click, enter password.
   Keychain installs cert + private key together. Skip step 1 there.
