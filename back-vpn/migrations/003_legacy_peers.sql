CREATE TABLE legacy_peers (
 public_key TEXT PRIMARY KEY,
 allowed_ips_json TEXT NOT NULL,
 psk_ciphertext BLOB NOT NULL,
 psk_nonce BLOB NOT NULL,
 encryption_version TEXT NOT NULL,
 keepalive INTEGER NOT NULL CHECK(keepalive BETWEEN 0 AND 65535),
 imported_at TEXT NOT NULL
);
