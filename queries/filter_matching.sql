-- name: FindMatchingFilters :many
WITH matched AS (
    SELECT f.id, f.response_type
    FROM filter_triggers t
    JOIN filters f ON t.filter_id = f.id
    WHERE f.chat_id = ?1
    AND INSTR(' ' || ?2 || ' ', ' ' || t.trigger_text || ' ') > 0
)
SELECT m.id, m.response_type, COALESCE(ft.response_text, '') AS response_text,
       COALESCE(fm.media_hash, '') AS media_hash, COALESCE(fm.media_type, '') AS media_type, 
       COALESCE(fr.reaction, '') AS reaction
FROM matched m
LEFT JOIN filter_text_resp ft ON m.id = ft.filter_id
LEFT JOIN filter_media_resp fm ON m.id = fm.filter_id
LEFT JOIN filter_reaction_resp fr ON m.id = fr.filter_id;
