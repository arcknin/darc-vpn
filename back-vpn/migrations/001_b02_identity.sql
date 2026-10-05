CREATE TABLE administrators (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'blocked')),
    created_at TEXT NOT NULL
);

CREATE TABLE accounts (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'blocked')),
    created_at TEXT NOT NULL,
    primary_profile_id TEXT NOT NULL UNIQUE REFERENCES profiles(id) DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE profiles (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    display_name TEXT NOT NULL,
    is_primary INTEGER NOT NULL CHECK (is_primary IN (0, 1)),
    endpoint TEXT NOT NULL CHECK (endpoint = 'vpn.darcanix.com'),
    peer_public_key TEXT NOT NULL UNIQUE,
    address TEXT NOT NULL,
    allowed_ips_json TEXT NOT NULL,
    dns_json TEXT NOT NULL,
    private_ciphertext BLOB NOT NULL,
    private_nonce BLOB NOT NULL,
    encryption_version TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE UNIQUE INDEX one_primary_profile_per_account
    ON profiles(account_id) WHERE is_primary = 1;
CREATE INDEX profiles_by_account ON profiles(account_id, created_at, id);

CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    token_digest BLOB NOT NULL UNIQUE CHECK (length(token_digest) = 32),
    account_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('friend', 'admin')),
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX sessions_by_account ON sessions(account_id);

CREATE TABLE terms_acceptances (
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    version TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    accepted_at TEXT NOT NULL,
    event_id TEXT NOT NULL UNIQUE,
    PRIMARY KEY (account_id, version, sha256)
);

CREATE TABLE revocation_operations (
    account_id TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('pending', 'failed', 'completed')),
    error TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL
);

CREATE TABLE revocation_peers (
    account_id TEXT NOT NULL REFERENCES revocation_operations(account_id) ON DELETE CASCADE,
    profile_id TEXT NOT NULL,
    peer_id TEXT NOT NULL,
    removed INTEGER NOT NULL DEFAULT 0 CHECK (removed IN (0, 1)),
    error TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (account_id, profile_id)
);
