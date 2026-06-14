-- Remove any orphan shell sizes that didn't get mapped during backfill (if any)
DELETE FROM WORK_ORDER_SHELL_SIZE WHERE ID_SIZE IS NULL;

-- Set ID_SIZE to NOT NULL as it is now required
ALTER TABLE WORK_ORDER_SHELL_SIZE ALTER COLUMN ID_SIZE SET NOT NULL;

-- Drop the old raw string SIZE column
ALTER TABLE WORK_ORDER_SHELL_SIZE DROP COLUMN SIZE;
