-- Tenant display names are not identifiers: two organisations (or two
-- households of the same family name) may share one. Uniqueness made the
-- second signup fail with a 500. The slug stays UNIQUE and is the handle.
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_name_key;
