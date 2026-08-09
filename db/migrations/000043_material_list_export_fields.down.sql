ALTER TABLE MATERIAL_LIST_ITEM
    DROP CONSTRAINT IF EXISTS fk_material_list_item_qty_wo_size,
    DROP CONSTRAINT IF EXISTS fk_material_list_item_qty_wo_shell,
    DROP CONSTRAINT IF EXISTS chk_material_list_item_qty_wo_scope_references,
    DROP CONSTRAINT IF EXISTS chk_material_list_item_qty_wo_scope,
    DROP CONSTRAINT IF EXISTS chk_material_list_item_cons_per_pc,
    DROP CONSTRAINT IF EXISTS chk_material_list_item_category;

ALTER TABLE MATERIAL_LIST_ITEM
    DROP COLUMN IF EXISTS ID_QTY_WO_SIZE,
    DROP COLUMN IF EXISTS ID_QTY_WO_SHELL,
    DROP COLUMN IF EXISTS QTY_WO_SCOPE,
    DROP COLUMN IF EXISTS CONS_PER_PC,
    DROP COLUMN IF EXISTS CATEGORY;
