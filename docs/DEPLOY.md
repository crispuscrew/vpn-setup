# Deploying a server

Step-by-step to stand up the stack from a bare VPS: the Marzneshin panel plus one
all-in-one node, then extra nodes, the config-as-code reconcile, and the delivery
bot. For the design behind it see the [README](../README.md); this is the runbook.

## What you end up with

- A **panel** host running Marzneshin plus a local all-in-one `marznode` that serves
  VLESS-Reality.
- Zero or more **extra node** hosts, each running a `marznode` the panel reaches over
  mutual TLS. A user's single subscription spans every node, and clients auto-select
  the fastest with failover.
- The `vpn` CLI reconciling panel services and users from `vpn.yaml`.
- The `vpnbot` Telegram bot handing each user their subscription URL and QR.

## Prerequisites

On your **control machine** (where you run the commands, not the server):

- `ansible-playbook` with the `ansible.posix` collection
  (`ansible-galaxy collection install ansible.posix`).
- Either Podman or Docker, for the `make` build of the Go binaries. The host Go
  toolchain is not required; the build runs in a pinned container.
- An SSH keypair whose public half you can put on the server.

Each **server**:

- A fresh VPS running Alma Linux 9 (or a RHEL 9 clone), reachable by SSH as `root`
  with your key.
- For a real deployment, a domain name pointed at the panel host (optional but
  recommended; without one, subscription links are IP-only and use plain HTTP).

## 1. Get the code and build the tools

```
git clone https://github.com/crispuscrew/vpn-setup.git
cd vpn-setup
make build          # builds bin/vpn and bin/vpnbot in the container
```

## 2. Write the inventory

Real inventories are gitignored; copy an example and fill it in.

Single all-in-one box:

```
cp ansible/inventory/single.yml.example ansible/inventory/prod.yml
```

Set `ansible_host` to your VPS IP, `ansible_ssh_private_key_file` to your key, and
`reality_sni` to a reachable TLS 1.3 site that is not blocked in your target region
(`dl.google.com`, `www.google.com`, or `www.cloudflare.com` are verified good;
**do not** use `www.microsoft.com`). Set `panel_domain` to your FQDN, or leave it
empty for an IP-only test box.

## 3. Provision the host

```
ansible-playbook ansible/site.yml -i ansible/inventory/prod.yml
```

This installs the container engine, brings up the pinned panel
(`dawsh/marzneshin:v0.7.4`) and node (`dawsh/marznode:v0.5.7`), renders the node's
Reality inbound, opens the firewall, and registers the node with the panel. It
generates a sudo admin once and stores its password on your control machine under
`ansible/.secrets/panel_admin_<host>.txt` (gitignored); re-runs reuse it, so the API
password stays stable. The play waits until the node reports healthy before
finishing.

Reach the panel at `http://<panel-ip>:8000` (or `https://<domain>` once TLS is set
up below). Swagger is at `/docs` while `panel_docs` is true.

### HTTPS (recommended for real use)

Set `panel_domain` in the inventory to a hostname whose DNS A-record points at the
panel host, then re-run `site.yml`. When a domain is set, the playbook runs a
Caddy reverse proxy on the panel host that gets an automatic Let's Encrypt
certificate, terminates HTTPS on 443, and proxies to the panel over loopback; the
public plain-HTTP port is closed. Subscription links switch to
`https://<domain>/sub/...` automatically. Any hostname works, including a free one
(for example a `*.duckdns.org` subdomain). Point the `vpn` CLI and the bot at
`https://<domain>` instead of the IP once this is on.

Because the all-in-one panel host also runs a node whose Reality inbound defaults
to 443, and Caddy needs 443 for HTTPS, set that host's `reality_port` to another
port (for example 8443) in the inventory. The play refuses to continue otherwise.
Dedicated nodes keep 443.

## 4. Apply the panel config-as-code

`vpn` reads the panel URL and admin credentials from the environment, never from
files, then reconciles the declared state in `vpn.yaml` (a service grouping every
discovered inbound, plus any declared users). Reconcile is idempotent and additive.

```
export VPN_PANEL_URL=http://<panel-ip>:8000
export VPN_PANEL_USERNAME=admin
export VPN_PANEL_PASSWORD="$(cat ansible/.secrets/panel_admin_<host>.txt)"

./bin/vpn apply -f vpn.yaml     # create/update services + users
./bin/vpn status                # discovered inbounds, services, users
./bin/vpn health                # panel + per-node health (non-zero exit if degraded)
./bin/vpn sub <user>            # print a user's subscription URL
```

## 5. Add another node

Use the multi-node inventory: a `panel` host plus a `nodes` group. Give each host a
`location` (for example "Serbia"); it names the panel node and, with an optional
`node_label` like "🇷🇸 Serbia", the name users see in their client. Extra nodes may
run Debian or Alma; the panel host stays Alma.

```
cp ansible/inventory/multi.yml.example ansible/inventory/prod.yml
# set ansible_host + location (and node_label) per host, then:
ansible-playbook ansible/site.yml -i ansible/inventory/prod.yml
```

The panel opens the node's gRPC port to itself only, fetches its own client
certificate, and connects over mutual TLS; the node's subscription host is pointed
at the node's own public address. Re-run `./bin/vpn status` and confirm the new
node's inbound appears.

To let users pick this location, add a per-location service to `vpn.yaml` and
re-apply:

```
  - name: Serbia
    nodes: ["Serbia"]     # every inbound on the node named Serbia
```

The `all` service (`inbounds: ["*"]`) always spans every node; per-location services
scope a user to one node. The bot's `/add` picker grants any subset of these.

### Protocols

Each node exposes five protocols, all folded into the one subscription:

- **VLESS+Reality** (TCP) - needs no domain or cert.
- **Hysteria2** (UDP, port 8443) - `hysteria2_enabled`.
- **TUIC** (UDP, port 8444, via sing-box) - `tuic_enabled`.
- **Trojan** (TCP, port 8445, via xray) - `trojan_enabled`.
- **Shadowsocks** (TCP+UDP, port 8388, via xray) - `shadowsocks_enabled`. Plaintext
  AEAD (its own encryption), so it needs no TLS or domain; the cipher is chosen per
  user by the panel.

Hysteria2, TUIC, and Trojan all need TLS but the nodes carry no per-node domain, so
they share one per-node self-signed cert and their subscription links set
`allowinsecure` (see the note in `group_vars`). Every protocol is toggled per host
(default on) and discovered by the panel automatically. A client that reads the
subscription (Hiddify, sing-box, v2rayN) offers all of them and auto-selects the
fastest.

After enabling a new protocol on nodes, run `vpn apply` once so the services include
the newly discovered inbounds (the `all` and per-location services pick them up).

## 6. Run the delivery bot

Create a bot with @BotFather for the token, and get your numeric Telegram id (for
example from @userinfobot) for the admin allowlist.

The recommended path runs the bot as a container on the panel host, managed by the
same playbook, so delivery no longer depends on an operator's laptop. Put the token
in a gitignored secret file on the control node and opt the panel host in:

```
printf '%s' '<botfather-token>' > ansible/.secrets/vpnbot_token.txt
```

```yaml
# inventory, on the panel host
vpnbot_enabled: true
vpnbot_admins: "123456789"      # comma-separated admin Telegram ids
```

Re-run the playbook. The `vpnbot` role builds the small non-root image from source
on the host, then runs it host-networked so it reaches the panel over loopback
(`http://127.0.0.1:{{ panel_port }}`, no public port, no TLS hop) and the AWG nodes
over SSH. The panel credentials and `VPNBOT_AWG_NODES` are wired from the inventory;
the delivery ledger persists in `/var/lib/vpnbot`. For AWG the playbook mints a
dedicated bot key and authorises it on each node restricted to the peer agent (see
step 7), so a bot compromise cannot get a shell.

To run the bot off-host instead (for example on a laptop while testing), build the
container with `make image` and run it with a writable `/state` volume and these set:

- `VPNBOT_TOKEN` bot token from @BotFather.
- `VPNBOT_ADMINS` comma-separated admin Telegram user ids.
- `VPN_PANEL_URL`, `VPN_PANEL_USERNAME`, `VPN_PANEL_PASSWORD` as in step 4.
- Optional `VPNBOT_LEDGER` (default `/state/ledger.json`).

An admin runs `/add <username>`, taps the locations that user may reach, then Done
to get a one-time claim link; the recipient taps it and receives their subscription
URL and QR once, with a `/setup` device guide. `/list`, `/revoke`, and `/help` round
out the admin face.

## 7. AmneziaWG (optional)

AmneziaWG is an obfuscated-WireGuard protocol that cannot ride the Marzneshin
subscription, so it runs as a standalone userspace server per node and is delivered
out of band as a `.conf` the AmneziaVPN app imports.

Enable the server per host in the inventory:

```
awg_enabled: true    # stands up the amneziawg server on this host (51820/udp)
```

Re-run the playbook; the `amneziawg` role builds the pinned server image, generates a
per-node obfuscation profile, brings up `awg0`, and installs a node-side `awg-peer`
agent the bot drives over SSH.

When the bot runs on the panel host (step 6), this is wired automatically: the
playbook mints a dedicated bot SSH key, authorises it on each AWG node restricted to
the peer agent with a forced command (a bot compromise can only manage peers, never
get a shell), and passes the node list and key into the container. Nothing to set by
hand.

Running the bot off-host, set instead:

- `VPNBOT_AWG_NODES` - `Location=host` pairs, one per AWG node, e.g.
  `Estonia=203.0.113.1,Serbia=203.0.113.2`. The location must match the panel
  service/node name.
- `VPNBOT_SSH_KEY` (default `~/.ssh/amnezia-ansible`), `VPNBOT_SSH_USER` (default
  `root`) - how the bot reaches the node agents. The bot must run on a host that can
  SSH to the nodes.

A user runs `/awg`, picks one of their granted locations, and receives an importable
`.conf` plus a QR. Only a location the user is granted and that has an AWG node
configured is offered, so per-location access still applies. Peers are recorded so a
repeat `/awg` re-sends the same config.

## Security

- Never commit secrets. The bot token, panel admin password
  (`ansible/.secrets/`), and node certificates are gitignored and environment-only.
- Access is admin-provisioned: users exist only when created in the panel or via the
  bot's admin `/add`, and the bot delivers only to the claiming recipient.
- Image tags are pinned in `versions.yml`; never move to `:latest`.

## Troubleshooting

- **Reality clients fail to connect** with "processed invalid connection": the
  `reality_sni` target is not borrowable. Switch to `dl.google.com` and re-run the
  playbook; the change propagates to the inbound and the generated links.
- **Rootless Podman can't resolve names** behind a VPN with DNS-leak protection: add
  `--dns=1.1.1.1` (or your trusted resolver) to the build. A stopgap, not a fix.
- **A node never goes healthy**: confirm the panel can reach the node on
  `marznode_port` (53042) and that the node host's firewall allows the panel.
