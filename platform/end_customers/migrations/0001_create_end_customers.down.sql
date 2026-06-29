-- ============================================================
-- Migration  : 0001_create_end_customers (DOWN)
-- Description: Drop the end_customers table and its indexes.
-- ============================================================

DROP INDEX  IF EXISTS idx_end_customers_search;
DROP INDEX  IF EXISTS idx_end_customers_cursor;
DROP INDEX  IF EXISTS idx_end_customers_tenant_status;
DROP TABLE  IF EXISTS end_customers;
