-- placeholder so sqlc has something to parse
-- name: SDEBuildNumber :one
SELECT value FROM config WHERE key = $1;
