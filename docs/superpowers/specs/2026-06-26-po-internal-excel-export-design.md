# PO Internal Excel Export Design

## Goal

Provide a dedicated Excel export for `PO Internal` using `templates/exports/xlsx/template_po_internal.xlsx`, with download access from:

- `GET /api/v1/po-internals/:id/export/excel`
- FE list page `/po-internal`
- FE detail page `/po-internal/:id`

## Recommended Approach

Create a dedicated backend export use case, following the existing `Work Order` export pattern:

1. Load PO internal detail from existing transaction reads.
2. Load company profile from `profil_perusahaan` for the delivery address block.
3. Open `template_po_internal.xlsx`.
4. Fill header, supplier, company, and summary cells.
5. Expand the item table dynamically when item count exceeds the six rows available in the template.
6. Return the workbook as downloadable `.xlsx`.

## Data Mapping

### Supplier block

- `supplier_name`
- `supplier_addr`
- `supplier_contact`
- `supplier_email`
- `supplier_telp`
- `supplier_fax`

### Delivery address block

Taken from `profil_perusahaan`:

- `nama`
- `alamat`
- `email`
- `no_telp`

### Document block

- `Purchase Order No` from PO internal id/name formatter
- `Date` from `tanggal`
- `Name of PO` from `nama_po`
- `Currency` from `currency`
- `C.P.O` from `cpo`
- `Term` from `term`
- `Ship Date` from `ship_date`

### Item table

Per item row:

- `No`
- `Item`
- `Description`
- `Qty`
- `Unit`
- `Unit Price`
- `Total`

### Summary

- `Total` = sum of qty
- `SubTotal` = sum of item totals
- `PPn` = keep template placeholder/default for now
- `Balance` = subtotal
- `Terbilang` = blank in first pass

## Dynamic Row Strategy

Template capacity:

- item rows: `18..23`
- summary starts at row `24`

If item count is greater than six:

1. duplicate the blank item row template before the summary block
2. repeat until all items fit
3. write item values into the expanded rows
4. let totals/footer/signature rows shift downward automatically

## Frontend Integration

### List page

Add an `Export Excel` action beside `View`.

### Detail page

Add an `Export Excel` button in the top action group, matching the Work Order detail page pattern.

### API client

Add `downloadPOInternalExcel(id)` that:

- requests `blob`
- extracts file name from `Content-Disposition`
- returns `{ blob, fileName }`

## Verification

- `go test ./...`
- `make swag`
- `npm run build`
- smoke test download from backend endpoint
- smoke test FE buttons on list and detail pages
