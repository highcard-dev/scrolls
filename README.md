# Scrolls Registry

## keepAliveTraffic

On Kubernetes runtimes, `keepAliveTraffic` is an enforcement rule for job procedures. If the configured pod RX-byte window elapses below the expected traffic threshold, druid stops that procedure cleanly and lets the command run mode decide what runs next.

For Minecraft coldstarter scrolls, attach `keepAliveTraffic` to the real runtime procedure's `main` expected port. Do not attach it to the coldstarter procedure; the standby listener must remain available while idle.

Minecraft scrolls should use `10kb/30m` on the real runtime `main` port. Server-list pings are small and should not keep a runtime warm; active clients generate recurring keep-alive/play traffic and should keep the runtime alive.
