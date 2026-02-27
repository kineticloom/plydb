Template prompt for adding a new data source

---

I'd like to add [SOMEDATASOURCE] as a new type of data source.

Implementation notes:

- Docuentation for [SOMEDATASOURCE] is here: [WEBSITE_FOR_SOMEDATASOURCE]
- Data source config schema spec is in /specs/config_schema.md. It will need to
  be updated. A copy of the file is in
  /skills/plydb/references/config_schema.md, make the same changes there.
- Golang data structure for data source config is in queryengine/config.go
- queryengine.New will need to be updated (in queryengine/engine.go)
- check if queryengine.PreprocessQuery will need to be updated (in
  queryengine/preprocess.go)
- AutoScanProvider.Provide (in semanticcontext/scanner.go) may need to be
  updated
- Look for other code that should be updated
- Remember to add unit tests and integration tests. Check existing patterns.
- Update README.md
- Add new example doc in /examples/connect_to_[SOMEDATASOURCE]/ with simple
  example in pattern of /examples/connect_to_csv_files/ or
  /examples/connect_to_sqlite/
