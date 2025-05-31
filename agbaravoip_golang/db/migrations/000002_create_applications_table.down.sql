DROP TRIGGER IF EXISTS update_applications_updated_at ON applications;
-- Note: The update_updated_at_column() function is not dropped here
-- as it was created in the first migration and might be used by other tables.
-- It should only be dropped if no other table uses it, typically in the
-- 000001_create_accounts_table.down.sql or a dedicated function migration.

DROP TABLE IF EXISTS applications;
