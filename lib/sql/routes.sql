WITH
    content_item_routes AS (
        SELECT route ->> 'path' AS path, route ->> 'type' AS type
        FROM content_items AS c, LATERAL jsonb_array_elements(c.routes || c.redirects) AS route
    )
SELECT
    content_items.rendering_app AS backend,
    route ->> 'path' AS path,
    route ->> 'type' AS match_type,
    route ->> 'destination' AS destination,
    route ->> 'segments_mode' AS segments_mode,
    content_items.schema_name AS schema_name,
    CASE
        WHEN content_items.schema_name = 'gone' THEN content_items.details
        ELSE NULL
    END AS details
FROM content_items, LATERAL jsonb_array_elements(
        content_items.routes || content_items.redirects
    ) AS route
UNION ALL
SELECT
    publish_intents.rendering_app AS backend,
    route ->> 'path' AS path,
    route ->> 'type' AS match_type,
    route ->> 'destination' AS destination,
    route ->> 'segments_mode' AS segments_mode,
    NULL AS schema_name,
    NULL AS details
FROM publish_intents, LATERAL jsonb_array_elements(publish_intents.routes) AS route
WHERE
    NOT EXISTS (
        SELECT 1
        FROM content_item_routes cir
        WHERE
            cir.path = route ->> 'path'
            AND cir.type = route ->> 'type'
    );