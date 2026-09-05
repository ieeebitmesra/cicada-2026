-- 003_add_round_details.sql — Add description and solve limit configuration to rounds table

ALTER TABLE rounds ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE rounds ADD COLUMN limit_solves BOOLEAN DEFAULT 0;
ALTER TABLE rounds ADD COLUMN max_solves INTEGER DEFAULT 0;
