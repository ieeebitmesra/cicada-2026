-- 001_initial.sql — initial schema for the IEEE CTF submission portal

CREATE TABLE teams (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    UNIQUE NOT NULL,
    ssh_user    TEXT    UNIQUE NOT NULL,
    ssh_pass    TEXT    NOT NULL,           -- bcrypt hash
    pgp_pubkey  TEXT    NOT NULL DEFAULT '',
    registered  BOOLEAN DEFAULT 0,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE rounds (
    id          INTEGER PRIMARY KEY,
    name        TEXT    NOT NULL,
    points      INTEGER NOT NULL,
    flag_hash   TEXT    NOT NULL,           -- SHA-256 of correct flag
    is_active   BOOLEAN DEFAULT 1,
    sort_order  INTEGER NOT NULL
);

CREATE TABLE submissions (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    team_id      INTEGER NOT NULL REFERENCES teams(id),
    round_id     INTEGER NOT NULL REFERENCES rounds(id),
    flag_input   TEXT    NOT NULL,
    is_correct   BOOLEAN NOT NULL,
    submitted_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_correct_once
    ON submissions(team_id, round_id) WHERE is_correct = 1;

CREATE TABLE hint_usage (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    team_id     INTEGER NOT NULL REFERENCES teams(id),
    round_id    INTEGER NOT NULL REFERENCES rounds(id),
    hint_type   TEXT    NOT NULL CHECK(hint_type IN ('plain', 'encoded')),
    hint_index  INTEGER NOT NULL,
    pgp_proof   TEXT    NOT NULL,
    cost_points REAL    NOT NULL,
    used_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE skips (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    team_id     INTEGER NOT NULL REFERENCES teams(id),
    round_id    INTEGER NOT NULL REFERENCES rounds(id),
    cost_points REAL    NOT NULL,
    skipped_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(team_id, round_id)
);

CREATE VIEW scoreboard AS
SELECT
    t.id   AS team_id,
    t.name AS team_name,
    COALESCE(SUM(CASE WHEN s.is_correct THEN r.points ELSE 0 END), 0)
      - COALESCE((SELECT SUM(cost_points) FROM hint_usage WHERE team_id = t.id), 0)
      - COALESCE((SELECT SUM(cost_points) FROM skips      WHERE team_id = t.id), 0)
      AS total_score,
    COUNT(DISTINCT CASE WHEN s.is_correct THEN s.round_id END) AS rounds_solved
FROM teams t
LEFT JOIN submissions s ON s.team_id = t.id
LEFT JOIN rounds r      ON r.id = s.round_id
GROUP BY t.id
ORDER BY total_score DESC;
