# Scroll Validation Scripts

This directory contains validation scripts for the Druid scrolls repository.

## validate-scrolls.sh

Validates all `scroll.yaml` files in the repository.

### What it checks

- ✅ Required fields (`name`, `desc`, `app_version`)
- ✅ Port definitions (valid numbers, 1-65535 range)
- ✅ Sleep handler file existence
- ✅ Init command definition
- ✅ Commands section structure
- ✅ Procedures definition

### Usage

```bash
# From repository root
./scripts/validate-scrolls.sh

# Results are written to /tmp/scroll-validation-results.txt
```

### Exit codes

- `0` - All scrolls valid
- `1` - One or more scrolls failed validation

### CI Integration

This script is automatically run by GitHub Actions on:
- Pull requests that modify scroll files
- Pushes to master branch

See `.github/workflows/validate-scrolls.yml` for the workflow configuration.

## Common validation errors

### Missing sleep handler file

```
ERROR: Sleep handler not found: generic
```

**Fix:** Create the missing sleep handler file (e.g., `generic.lua`) or update the scroll to reference an existing handler.

### Invalid port number

```
ERROR: Port number out of range: 99999
```

**Fix:** Port numbers must be between 1 and 65535.

### Missing required field

```
ERROR: Missing required field: app_version
```

**Fix:** Add the missing field to the `scroll.yaml` file.

## Warnings (non-blocking)

Warnings don't fail the validation but should be reviewed:

- `No mandatory ports defined` - Consider adding `mandatory: true` to at least one port
- `No ports defined` - Scroll has no network ports (intentional for some utility scrolls)
- `No init command defined` - Scroll may not start properly
