-- Drop in reverse order of .up.sql to respect foreign keys.

DROP TABLE IF EXISTS bookmarks;
DROP TABLE IF EXISTS reactions;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS post_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS oauth_accounts;
DROP TABLE IF EXISTS users;

-- No DROP EXTENSION: extensions are database-scoped and another schema may rely
-- on them. Dropping the tables is enough to reverse this migration.
