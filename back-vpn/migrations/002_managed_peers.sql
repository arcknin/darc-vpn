CREATE TABLE peer_intents (
 profile_id TEXT PRIMARY KEY REFERENCES profiles(id),
 ipv4 TEXT NOT NULL UNIQUE,
 ipv6 TEXT NOT NULL UNIQUE,
 state TEXT NOT NULL CHECK(state IN ('pending','ready','failed','revoked')),
 error TEXT NOT NULL DEFAULT '',
 revision INTEGER NOT NULL DEFAULT 1,
 updated_at TEXT NOT NULL
);
