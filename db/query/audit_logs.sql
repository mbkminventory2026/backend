-- name: CreateAuditLog :one
INSERT INTO AUDIT_LOGS (
    actor_user_id,
    actor_username,
    actor_role,
    action,
    module,
    entity_type,
    entity_id,
    entity_label,
    method,
    route,
    before_data,
    after_data,
    changed_fields
) VALUES (
    sqlc.narg(actor_user_id),
    sqlc.arg(actor_username),
    sqlc.arg(actor_role),
    sqlc.arg(action),
    sqlc.arg(module),
    sqlc.arg(entity_type),
    sqlc.arg(entity_id),
    sqlc.arg(entity_label),
    sqlc.arg(method),
    sqlc.arg(route),
    sqlc.narg(before_data),
    sqlc.narg(after_data),
    sqlc.arg(changed_fields)
)
RETURNING *;

-- name: ListAuditLogs :many
SELECT *
FROM AUDIT_LOGS
WHERE (
    sqlc.arg(search_term)::text = '' OR
    actor_username ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    actor_role ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    module ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    entity_type ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    entity_label ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    route ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
AND (
    sqlc.arg(action_filter)::text = '' OR
    action = sqlc.arg(action_filter)::text
)
AND (
    sqlc.arg(module_filter)::text = '' OR
    module = sqlc.arg(module_filter)::text
)
AND (
    sqlc.arg(entity_type_filter)::text = '' OR
    entity_type = sqlc.arg(entity_type_filter)::text
)
AND (
    sqlc.arg(actor_user_id_filter)::int = 0 OR
    actor_user_id = sqlc.arg(actor_user_id_filter)::int
)
AND (
    sqlc.arg(date_from)::timestamptz IS NULL OR
    created_at >= sqlc.arg(date_from)::timestamptz
)
AND (
    sqlc.arg(date_to)::timestamptz IS NULL OR
    created_at <= sqlc.arg(date_to)::timestamptz
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'actor_username' AND NOT sqlc.arg(sort_desc)::bool THEN actor_username END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'actor_username' AND sqlc.arg(sort_desc)::bool THEN actor_username END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'action' AND NOT sqlc.arg(sort_desc)::bool THEN action END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'action' AND sqlc.arg(sort_desc)::bool THEN action END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'module' AND NOT sqlc.arg(sort_desc)::bool THEN module END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'module' AND sqlc.arg(sort_desc)::bool THEN module END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'entity_type' AND NOT sqlc.arg(sort_desc)::bool THEN entity_type END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'entity_type' AND sqlc.arg(sort_desc)::bool THEN entity_type END DESC,
    created_at DESC,
    id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountAuditLogs :one
SELECT COUNT(*)
FROM AUDIT_LOGS
WHERE (
    sqlc.arg(search_term)::text = '' OR
    actor_username ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    actor_role ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    module ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    entity_type ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    entity_label ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    route ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
AND (
    sqlc.arg(action_filter)::text = '' OR
    action = sqlc.arg(action_filter)::text
)
AND (
    sqlc.arg(module_filter)::text = '' OR
    module = sqlc.arg(module_filter)::text
)
AND (
    sqlc.arg(entity_type_filter)::text = '' OR
    entity_type = sqlc.arg(entity_type_filter)::text
)
AND (
    sqlc.arg(actor_user_id_filter)::int = 0 OR
    actor_user_id = sqlc.arg(actor_user_id_filter)::int
)
AND (
    sqlc.arg(date_from)::timestamptz IS NULL OR
    created_at >= sqlc.arg(date_from)::timestamptz
)
AND (
    sqlc.arg(date_to)::timestamptz IS NULL OR
    created_at <= sqlc.arg(date_to)::timestamptz
);

-- name: GetAuditLogByID :one
SELECT *
FROM AUDIT_LOGS
WHERE id = $1
LIMIT 1;
