DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='ratio_size_marker' AND column_name='ratio_plan') THEN
        ALTER TABLE RATIO_SIZE_MARKER RENAME COLUMN RATIO_PLAN TO QTY_PLAN;
    END IF;
END $$;
