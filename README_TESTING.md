## Scroll Testing

This repository includes automated testing for all scrolls to ensure they start correctly and open their mandatory ports.

### Test Script

The `test-scrolls.sh` script tests each scroll by:

1. **Finding all scrolls** - Locates all `scroll.yaml` files
2. **Parsing port configuration** - Extracts port numbers from scroll.yaml
3. **Starting druid serve** - Runs the scroll with `druid serve`
4. **Checking ports** - Verifies each port opens within timeout (60s)
5. **Cleanup** - Stops druid process
6. **Reporting** - Logs results to `test-results/`

### Running Tests Manually

#### Prerequisites

- `druid` CLI installed (or set `DRUID_BIN` env var)
- `netcat` (nc) for port checking
- Nix (for dependency resolution)

#### Run All Tests

```bash
./test-scrolls.sh
```

#### Run Specific Scroll

```bash
cd scrolls/minecraft/papermc/1.21.7
druid serve
# In another terminal:
nc -zv localhost 25565
```

### Environment Variables

- `DRUID_BIN` - Path to druid binary (default: `druid`)
- `TIMEOUT_SECONDS` - Port check timeout (default: 60)

### Test Results

Results are saved to `test-results/`:
- Logs for each scroll test
- Named: `{game}_{variant}_{version}.log`

### GitHub Actions

Tests can be run manually via GitHub Actions:

1. Go to **Actions** tab
2. Select **Test Scrolls** workflow
3. Click **Run workflow**

**Note:** Testing all 222 scrolls takes ~3 hours.

### Future: PR CI Integration

Once stable, uncomment in `.github/workflows/test-scrolls.yml`:

```yaml
on:
  pull_request:
    paths:
      - 'scrolls/**'
```

This will automatically test changed scrolls on PRs.

### Troubleshooting

**Port check timeout:**
- Some servers need longer to start
- Check logs in `test-results/`
- Increase `TIMEOUT_SECONDS` if needed

**druid serve fails:**
- Check scroll.yaml syntax with `druid scroll validate`
- Ensure dependencies are available (Nix)
- Review logs in `test-results/`

**netcat not found:**
```bash
# Ubuntu/Debian
sudo apt-get install netcat-openbsd

# macOS
brew install netcat
```
