-- migrations/002_security_hardening.sql — Security and Concurrency Hardening

-- 1. Enforce unique hint redemption per team, round, type, and index (SEC-04)
CREATE UNIQUE INDEX IF NOT EXISTS idx_hint_usage_unique
    ON hint_usage(team_id, round_id, hint_type, hint_index);

-- 2. Enforce uniqueness of PGP cryptographic proofs to prevent token replay (SEC-04)
CREATE UNIQUE INDEX IF NOT EXISTS idx_hint_usage_proof
    ON hint_usage(pgp_proof);

-- 3. Update scoreboard view with deterministic tie-breaking by earliest solve timestamp (SEC-14)
DROP VIEW IF EXISTS scoreboard;
CREATE VIEW scoreboard AS
SELECT
    t.id   AS team_id,
    t.name AS team_name,
    COALESCE(SUM(CASE WHEN s.is_correct THEN r.points ELSE 0 END), 0)
      - COALESCE((SELECT SUM(cost_points) FROM hint_usage WHERE team_id = t.id), 0)
      - COALESCE((SELECT SUM(cost_points) FROM skips      WHERE team_id = t.id), 0)
      AS total_score,
    COUNT(DISTINCT CASE WHEN s.is_correct THEN s.round_id END) AS rounds_solved,
    COALESCE(MAX(CASE WHEN s.is_correct THEN s.submitted_at END), t.created_at) AS last_solve_at
FROM teams t
LEFT JOIN submissions s ON s.team_id = t.id
LEFT JOIN rounds r      ON r.id = s.round_id
GROUP BY t.id
ORDER BY total_score DESC, last_solve_at ASC, team_name ASC;
