-- ============================================================
-- Migration  : 0002_create_end_customer_tags (DOWN)
-- Description: Drop the end_customer_tags table.
-- ============================================================

DROP INDEX IF EXISTS idx_end_customer_tags_by_tag;
DROP TABLE IF EXISTS end_customer_tags;
