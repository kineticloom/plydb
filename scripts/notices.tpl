# Third-Party Notices

PlyDB depends on the following open-source packages. Their licenses are reproduced below.

---
{{ range . }}
## {{ .Name }} ({{ .Version }})

**License:** [{{ .LicenseName }}]({{ .LicenseURL }})

```
{{ .LicenseText }}
```

---
{{ end }}
