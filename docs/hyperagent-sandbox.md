# Go modules in the Hyperagent sandbox

Audience: any agent or human building Balaur inside a Hyperagent
sandbox/container (and, generally, behind any TLS-intercepting proxy whose
minted certificates Go rejects). Nothing in this document is product code —
the Balaur binary and its users are unaffected.

## Symptom

`go mod download`, `go build`, `go vet`, `go test` fail with:

```
tls: failed to verify certificate: x509: certificate signed by unknown authority
```

…while `curl`, Python, and Node reach the very same URLs without complaint.

## Root cause (diagnosed June 2026)

- All sandbox egress is forced through a TLS-intercepting proxy
  (`HTTPS_PROXY` env). It mints a short-lived per-host leaf certificate
  signed by "HyperAgent Network Proxy CA (production)".
- That CA **is** on disk (`/usr/local/share/ca-certificates/`) and **is** in
  the system bundle, and the minted leaf's signature **is** cryptographically
  valid against it.
- But the minted leaf's **Issuer DN is encoded with different ASN.1 string
  types than the CA's Subject DN**. Go's chain builder finds candidate
  parents by exact byte equality of `leaf.RawIssuer` and `pool.RawSubject` —
  the encodings differ, so Go never even considers the CA, and reports
  "unknown authority". OpenSSL-based stacks (curl, Python, Node) compare
  decoded names tolerantly, which is why everything except Go works.

Verified with a stdlib probe against `https://proxy.golang.org`:

```go
leaf.CheckSignatureFrom(root)                  // <nil> — signature is valid
bytes.Equal(leaf.RawIssuer, root.RawSubject)   // false — Go can't chain it
```

Consequence: **no amount of CA-bundle plumbing fixes Go.** `SSL_CERT_FILE`,
`SSL_CERT_DIR`, or a hand-built `x509.CertPool` containing the exact
presented root all fail identically, because the failure happens at parent
*lookup*, before signature verification.

## What does NOT work (do not burn time retrying)

- Appending the proxy CA to `SSL_CERT_FILE` or a custom root pool — see above.
- `GOPROXY=direct` — every HTTPS host gets the same MITM treatment.
- `GOFLAGS=-insecure` — only affects direct VCS fetches, not proxy TLS.
- `GONOSUMDB` / `GONOSUMCHECK` — checksum layer, unrelated to TLS.

## The fix: a localhost GOPROXY shim

`127.0.0.1` is in `no_proxy`, so Go can talk **plain HTTP** to localhost.
Run a tiny forwarder that fetches from `https://proxy.golang.org` using
Python's OpenSSL-backed TLS (which tolerates the DN encoding):

```bash
python3 scripts/goproxy-shim.py &        # serves http://127.0.0.1:8099
export GOPROXY=http://127.0.0.1:8099
go mod download
```

Module integrity is **not** weakened: `GOSUMDB` stays enabled, and the
sum.golang.org checksum queries route through the same shim (`/sumdb/…`
paths are forwarded verbatim). Only the transport hop is swapped.

## Full sandbox bootstrap

```bash
# 1. Install Go (curl is unaffected by the cert issue; verify the sha256
#    from https://go.dev/dl/?mode=json):
curl -sfLO https://go.dev/dl/go1.26.4.linux-amd64.tar.gz
echo "<sha256>  go1.26.4.linux-amd64.tar.gz" | sha256sum -c -
sudo tar -C /usr/local -xzf go1.26.4.linux-amd64.tar.gz

# 2. Environment (per shell):
export PATH=$PATH:/usr/local/go/bin
export GOPROXY=http://127.0.0.1:8099
export CGO_ENABLED=0

# 3. Shim (once per sandbox boot):
python3 scripts/goproxy-shim.py >/tmp/goproxy-shim.log 2>&1 &
```

## Sandbox memory limits

The sandbox has ~4 GB RAM; an unbounded parallel build of Balaur's
dependency tree has been OOM-killed (SIGKILL, exit 137). When memory is
tight, bound the parallelism:

```bash
go build -p 2 ./...
go vet -p 2 ./...
go test -p 1 ./...
```

## Scope note

The proxy quirk may get fixed upstream at any time. Before reaching for the
shim, try a plain `go mod download` first — if it succeeds, none of this is
needed. The shim adds no value outside sandboxes with this interception
behavior.
