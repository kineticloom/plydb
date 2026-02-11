## Testing

Run unit tests for the a package

```
go test ./somepackage/...
```

Run integration tests for a package (requires Docker)

```
go test -tags=integration -v -timeout 300s ./somepackage/...
```
