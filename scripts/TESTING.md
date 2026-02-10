# Scroll Testing

This directory contains testing scripts for Druid scrolls.

## Test Levels

### 1. Static Validation (`validate-scrolls.sh`)

**What it tests:**
- YAML structure
- Required fields
- Port definitions
- Sleep handler file existence

**Speed:** ~30 seconds for all 222 scrolls  
**Use:** Quick pre-commit checks, CI on every PR

**Run locally:**
```bash
./scripts/validate-scrolls.sh
```

### 2. Runtime Testing (`test-scroll-runtime.sh`)

**What it tests:**
- Actually starts the server with `druid serve`
- Waits for mandatory ports to open
- Verifies server is truly running (not just configuration valid)
- Checks networking and dependencies

**Speed:** ~3 minutes per scroll (would take 11 hours for all 222)  
**Use:** Deep testing of critical scrolls, manual testing, scheduled nightly runs

**Run locally:**
```bash
# Requires druid CLI installed
./scripts/test-scroll-runtime.sh

# Test specific scroll
cd scrolls/minecraft/papermc/1.21.7
druid serve --port 8081
# In another terminal:
ss -ltn | grep :25565  # Check if main port is open
```

## GitHub Actions Workflows

### validate-scrolls.yml
- Runs on every PR
- Tests all scrolls with static validation
- Fast feedback (30 seconds)
- Blocks merge if validation fails

### test-scrolls-runtime.yml
- Runs on PR + manual trigger
- Tests representative sample (one per game type)
- ~10-15 minutes total (parallelized)
- Optional: Can be triggered manually with custom scroll pattern

## Testing Strategy

**Why not test all 222 scrolls in CI?**

Testing all scrolls with runtime checks would take 11+ hours:
- 222 scrolls × 3 minutes timeout = 666 minutes = 11 hours

**Current approach:**
1. **Static validation** (all scrolls, every PR) - Fast, catches most issues
2. **Runtime testing** (sample set, CI) - Proves concept, catches real failures
3. **Full runtime testing** (manual/scheduled) - Can be run weekly or on-demand

**Sample scrolls tested:**
- minecraft/minecraft-vanilla/1.21.7
- minecraft/papermc/1.21.7
- minecraft/minecraft-spigot/1.21.7
- minecraft/forge/1.21.7
- lgsm/cs2server
- rust/rust-vanilla/latest

These represent:
- All major Minecraft variants
- Source engine games (CS2)
- Rust servers
- Different dependency stacks

## Manual Full Testing

To test all scrolls manually:

```bash
# Set DRUID_CLI path if not in PATH
export DRUID_CLI=/path/to/druid

# Run full test suite (will take hours)
./scripts/test-scroll-runtime.sh

# Test specific game type
cd scrolls/minecraft/papermc
for dir in */; do
  cd "$dir"
  druid serve --port 8081 &
  DRUID_PID=$!
  sleep 60  # Wait for startup
  ss -ltn | grep :25565 && echo "✓ $dir" || echo "✗ $dir"
  kill $DRUID_PID
  cd ..
done
```

## CI Architecture

```
PR → [Static Validation] → Pass → Merge
         ↓ (if fails)
      Block PR

Manual/Scheduled → [Runtime Testing] → [Sample Scrolls] → Report
```

## Future Improvements

1. **Test changed scrolls only** - Runtime test only scrolls modified in PR
2. **Parallel matrix** - Split 222 scrolls into 10 jobs (1 hour each)
3. **Nightly full run** - Test all scrolls overnight, report failures
4. **Integration tests** - Test actual gameplay (connect to server, send commands)
5. **Performance tests** - Measure startup time, memory usage

## Debugging Failed Tests

### Static validation failure

```bash
# Check the error
cat /tmp/scroll-validation-results.txt

# Common issues:
# - Missing field: Add to scroll.yaml
# - Invalid port: Fix port number (1-65535)
# - Missing sleep handler: Add the .lua file
```

### Runtime test failure

```bash
# Check druid logs
tail -100 /tmp/druid-serve-<scroll-name>.log

# Common issues:
# - Missing dependencies: Add to scroll.yaml dependencies
# - Port conflict: Another service using the port
# - Timeout too short: Server needs longer to start
# - Failed download: Check URLs in scroll.yaml
```

### Manual debugging

```bash
cd scrolls/your-game/your-version
druid serve --port 8081

# In another terminal:
# Watch ports open:
watch -n 1 'ss -ltn | grep LISTEN'

# Check druid logs:
tail -f ~/.druid/logs/druid.log

# Check server process:
ps aux | grep your-game-server
```

## Contributing

When adding a new scroll:

1. ✅ **Validate** - Run `./scripts/validate-scrolls.sh` locally
2. ✅ **Test manually** - Start with `druid serve`, verify ports open
3. ✅ **Check mandatory ports** - Ensure `mandatory: true` for main game port
4. ✅ **Submit PR** - CI will validate automatically
5. ⏳ **Runtime test** - May be added to sample set if important

## Questions?

- **How long do tests take?** Static: 30s, Runtime sample: 10-15min, Full runtime: 11h
- **Which tests run on PRs?** Static validation only (fast)
- **How to add runtime test for my scroll?** Add to matrix in `test-scrolls-runtime.yml`
- **Why not test everything?** Too slow for CI, but can be done manually/scheduled
