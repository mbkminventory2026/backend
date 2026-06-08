Template export foundation for backend-generated documents.

Structure:
- xlsx/: Excel templates used by the Excel export renderer

Notes:
- Keep template filenames stable and lowercase with dashes.
- Recommended examples:
  - po-client-detail.xlsx
  - work-order-detail.xlsx
  - pr-internal-list.xlsx
- The backend resolves templates from EXPORT_TEMPLATE_DIR, which defaults to templates/exports.
