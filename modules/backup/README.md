# Backup Module

Backup and restore operations for directories and services.

## Configuration

The GCS bucket must be configured before use. The configuration is read in this order:

1. `config.local.yaml` (not committed to git) - **Recommended**
2. `config.yaml`
3. Environment variable: `CLY_BACKUP_GCS_BUCKET`

### Setup

1. Create `config.local.yaml` in the project root or `~/.config/cly/`:

```yaml
modules:
  backup:
    gcs_bucket: your-bucket-name
```

2. Or set environment variable:

```bash
export CLY_BACKUP_GCS_BUCKET=your-bucket-name
```

## Commands

### Gsync (visual sync)

```bash
cly gsync            # sync the workdir backup (modules.backup.source_dir -> gcs_bucket)
cly gsync sessions   # sync a named target from modules.backup.targets
cly gsync status     # list data from previous runs (per-target summary, --json)
```

Named targets are configured in `~/.config/cly/config.yaml` and can have any
number of sources; each source uploads under its `prefix` in the bucket:

```yaml
modules:
  backup:
    targets:
      sessions:
        gcs_bucket: syscd-sessions-a494
        sources:
          - source_dir: ~/.omp/agent/sessions
            prefix: omp
          - source_dir: ~/.pi/agent/sessions
            prefix: pi
```

`cly gsync status` reads `~/.local/state/cly/gsync/history.jsonl` — one JSON
line per run (time, target, uploaded/skipped/errors, seconds, mode). Flags:
`-n N` limit, `--target NAME` filter, `--json`.

On a terminal gsync shows the live TUI (syncing folders on top, finished ones
collapsed below, `/tmp` report at the end). Without a terminal (scheduled or
piped) it runs headless: no renderer, exits as soon as the sync finishes,
prints a one-line summary, and exits non-zero if any upload failed — so
`cly every --notify` (or any scheduler) can alert on failures.

Features:
- Automatic gcloud authentication check
- Parallel processing per folder and per file
- mtime + size diffing (interops with gsutil's `goog-reserved-file-mtime`)
- gitignore-style excludes in `~/.config/cly/gsyncignore`

### Backup Workdir

Backs up `~/Workdir` to GCS bucket:

```bash
cly backup workdir
```

Features:
- Automatic gcloud authentication check
- Parallel processing for faster uploads
- Excludes common artifacts (node_modules, __pycache__, .terraform, etc.)
- Keeps git history

### Download Backup

Downloads entire GCS backup as a compressed tar.gz archive:

```bash
cly backup download
```

With custom output path:

```bash
cly backup download -o my-backup.tar.gz
```

Features:
- Downloads to timestamped file by default: `workdir-backup-YYYYMMDD-HHMMSS.tar.gz`
- Shows file size after completion
- Automatic cleanup of temporary files

## Examples

```bash
# Backup workdir
cly backup workdir

# Download backup
cly backup download

# Download with custom filename
cly backup download -o workdir-$(date +%Y%m%d).tar.gz
```
