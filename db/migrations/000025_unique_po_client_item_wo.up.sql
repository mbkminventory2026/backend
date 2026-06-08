DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'unique_id_po_client_item'
    ) THEN
        ALTER TABLE WORK_ORDER ADD CONSTRAINT unique_id_po_client_item UNIQUE (ID_PO_CLIENT_ITEM);
    END IF;
END $$;
