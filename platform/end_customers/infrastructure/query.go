package infrastructure

// SQL queries for the end_customers module.
// Tags are loaded via a separate query (queryFindTags) and attached in Go.

const queryCreate = `
INSERT INTO end_customers
    (id, tenant_id, external_id, email, phone, first_name, last_name, status, metadata, created_at, updated_at)
VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
`

const queryFindByID = `
SELECT id, tenant_id, external_id, email, phone, first_name, last_name, status, metadata, created_at, updated_at, deleted_at
FROM end_customers
WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
`

const queryFindByEmail = `
SELECT id, tenant_id, external_id, email, phone, first_name, last_name, status, metadata, created_at, updated_at, deleted_at
FROM end_customers
WHERE tenant_id = $1 AND email = $2 AND deleted_at IS NULL
LIMIT 1
`

const queryFindByExternalID = `
SELECT id, tenant_id, external_id, email, phone, first_name, last_name, status, metadata, created_at, updated_at, deleted_at
FROM end_customers
WHERE tenant_id = $1 AND external_id = $2 AND deleted_at IS NULL
LIMIT 1
`

const queryUpdate = `
UPDATE end_customers SET
    external_id = $3,
    email       = $4,
    phone       = $5,
    first_name  = $6,
    last_name   = $7,
    status      = $8,
    metadata    = $9,
    updated_at  = $10
WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
`

const querySoftDelete = `
UPDATE end_customers SET
    status     = 'DELETED',
    deleted_at = NOW(),
    updated_at = NOW()
WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
`

const queryCount = `
SELECT COUNT(*) FROM end_customers
WHERE tenant_id = $1 AND deleted_at IS NULL
`

const queryCountByStatus = `
SELECT COUNT(*) FROM end_customers
WHERE tenant_id = $1 AND status = $2 AND deleted_at IS NULL
`

// queryFindTags retrieves all tags for a given end_customer_id.
const queryFindTags = `
SELECT tag FROM end_customer_tags WHERE end_customer_id = $1 ORDER BY tag
`

// queryReplaceTags deletes existing tags and inserts the new set.
// Called within the same transaction as queryCreate / queryUpdate.
const queryDeleteTags = `
DELETE FROM end_customer_tags WHERE end_customer_id = $1
`

const queryInsertTag = `
INSERT INTO end_customer_tags (end_customer_id, tag) VALUES ($1, $2)
ON CONFLICT DO NOTHING
`
