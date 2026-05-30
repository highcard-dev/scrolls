# Scrolls Registry

## keepAliveTraffic

On Kubernetes runtimes, `keepAliveTraffic` is an enforcement rule for job procedures. If the configured Hubble traffic window elapses with no observed flow for the expected port, druid stops that procedure cleanly and lets the command run mode decide what runs next.

For Minecraft coldstarter scrolls, attach `keepAliveTraffic` to the real runtime procedure's `main` expected port. Do not attach it to the coldstarter procedure; the standby listener must remain available while idle.
