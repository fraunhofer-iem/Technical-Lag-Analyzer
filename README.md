# Technical Lag

Technical lag is calculated based on the used version of each package and its newest available version.

The package information is taken from a CycloneDX Software Bill of Materials (SBOM), the version information
is queried from [deps.dev](www.deps.dev).

## Usage

```
 go run cmd/technicalLag.go --help
  -in string
        Path to SBOM
  -logLevel int
        Can be 0 for INFO, -4 for DEBUG, 4 for WARN, or 8 for ERROR. Defaults to INFO.
  -out string
        File to write the SBOM to
```

## Output