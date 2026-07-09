# vpn-setup

> **Status: early, pre-release.** Single-server direct access works end to end —
> Ansible stands up the panel plus one node serving VLESS-Reality, and the `vpn`
> CLI reconciles the panel as config-as-code. More protocols and the Telegram bot
> are in progress; interfaces may change until the first tagged release. This repo
> is the intended single source of truth for VPN deployment across sibling projects.

Automated, multi-protocol VPN deployment built on **[Marzneshin](https://github.com/marzneshin/marzneshin)**.
It stands up a Marzneshin panel plus one or more `marznode` nodes — which bundle
Xray, Hysteria2, and sing-box — to serve **VLESS+Reality, VMess, Trojan, Shadowsocks,
Hysteria2, TUIC, and ShadowTLS** from one place, and delivers each user's
**subscription URL + QR** over a Telegram bot. Direct access today; multi-hop is a
planned later layer.

## How it fits together

Two control surfaces, deliberately separated:

- **Protocols / inbounds** are declared as **core config files on each `marznode`**
  (`xray_config.json`, hysteria `config.yaml`, sing-box `config.json`). Each inbound
  tag is discovered by the panel over gRPC.
- **Services, Hosts, and Users** live in the panel and are driven through its **REST
  API** (`/api/*`, JWT from `POST /api/admins/token`). A user's single subscription
  URL — `https://<panel>/sub/<username>/<key>` — aggregates every protocol they can use.

Tooling reflects that split:

- **Ansible** (`ansible/`) bootstraps hosts, runs the pinned Marzneshin/marznode
  installers, and renders the node core config files.
- **Go** provides two binaries: `vpn` (`cmd/vpn`) reconciles the panel-side config
  through the REST API; `vpnbot` (`cmd/vpnbot`) is the Telegram delivery bot.

Pinned upstream image tags live in `versions.yml` (currently
`dawsh/marzneshin:v0.7.4` and `dawsh/marznode:v0.5.7`) — never `:latest`.

## Build

The build runs in a pinned container; the host Go toolchain is not required.

```
make            # list targets
make build      # build bin/vpn and bin/vpnbot
make lint       # go vet + gofmt check
make test       # go test ./...
```

## Deploy

Provision a host (panel plus one all-in-one node) with Ansible:

```
cp ansible/inventory/single.yml.example ansible/inventory/<name>.yml   # set host + reality_sni
ansible-playbook ansible/site.yml -i ansible/inventory/<name>.yml
```

## Operate

`vpn` drives the panel as config-as-code. It reads the panel URL and sudo-admin
credentials from the environment — never from files — then applies the declared
state in `vpn.yaml`:

```
export VPN_PANEL_URL=http://<host>:8000
export VPN_PANEL_USERNAME=admin VPN_PANEL_PASSWORD=…
vpn apply -f vpn.yaml     # reconcile services + users (idempotent)
vpn status                # discovered inbounds, services, users
vpn sub <user>            # print a user's subscription URL
```

## Delivery bot

`vpnbot` hands each user their subscription over Telegram. An admin runs
`/add <username>` — which creates the panel user and returns a one-time claim link;
the recipient taps it (`/start <code>`) and receives their subscription URL + QR
exactly once, tracked in a durable ledger and re-shown on demand. `/help`, `/list`,
and `/revoke <username>` round out the admin face. Every secret is environment-only:

```
make image                         # build the small non-root container
# then run it with these set (plus the VPN_PANEL_* credentials):
#   VPNBOT_TOKEN     bot token from @BotFather
#   VPNBOT_ADMINS    comma-separated admin Telegram user ids
# mount a writable volume at /state for the delivery ledger.
```

## Layout

- `cmd/vpn/` — operator CLI (panel REST reconcile: services, users, subscriptions).
- `cmd/vpnbot/` — Telegram delivery bot (subscription URL + QR, exactly-once) and its `Containerfile`.
- `internal/panel/` — typed Marzneshin REST client shared by the binaries.
- `internal/ledger/` — the bot's durable, atomically-written exactly-once delivery ledger.
- `ansible/` — host + node provisioning and node core-config templates.
- `vpn.yaml` — declared panel state applied by `vpn apply`.
- `versions.yml` — pinned upstream image tags.

## Security

Panel admin tokens, the bot token, and node certificates are secrets — they are
gitignored and must never be committed. Access is admin-provisioned: users are
created in the panel (UI or an admin bot command), and the bot only delivers a
subscription to an authorized recipient.

## License

TBD.
