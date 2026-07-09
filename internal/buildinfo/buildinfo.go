// Package buildinfo carries the tool's own version, stamped at build time.
package buildinfo

// Version is the vpn-setup tool version. It is "dev" for local builds and is
// overridden at release with `-ldflags "-X .../buildinfo.Version=<tag>"`.
var Version = "dev"
