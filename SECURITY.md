# Security policy

Lure is meant to be reached by untrusted clients, so its sensors are built to
give them nothing: each connection has a time limit and a read limit, nothing a
client sends is executed or forwarded, and login attempts always fail.

Keep the operator panel on a private address or set `-token`. Run the decoys on
a host you control and that holds nothing of value.

Report a suspected vulnerability privately through GitHub's security advisory
flow for this repository rather than a public issue. Include a minimal
reproduction and the affected version.
