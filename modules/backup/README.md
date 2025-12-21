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
