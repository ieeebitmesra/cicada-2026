-- 004_add_skip_text.sql — Add skip_text column to rounds table

ALTER TABLE rounds ADD COLUMN skip_text TEXT NOT NULL DEFAULT '';
