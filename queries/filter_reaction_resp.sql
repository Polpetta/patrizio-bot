-- name: InsertReactionResponse :exec
INSERT INTO filter_reaction_resp (filter_id, reaction)
VALUES (?1, ?2);

-- name: GetReactionResponseByFilterID :one
SELECT filter_id, reaction
FROM filter_reaction_resp
WHERE filter_id = ?1;
