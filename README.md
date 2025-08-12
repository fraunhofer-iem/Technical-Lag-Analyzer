# Technical Lag

Technical lag is calculated based on the used version of each package and its newest available version.
It is calculated as "libyears" as defined in [this article](https://ericbouwers.github.io/papers/icse15.pdf) by Joel Cox
et al. and as the version distance (how many releases are between the used version and the newest available version).

The package information is taken from a CycloneDX Software Bill of Materials (SBOM), the version information
is queried from [deps.dev](https://deps.dev).

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

## Docker Usage

You can build and run this application using Docker:

```bash
# Build the Docker image
docker build -t sbom-technical-lag .
```

### Working with Files

The Docker container is configured to make file input/output easy. The container's working directory is set to `/data`,
which is where you should mount your local directory containing input files and where output files will be written.

#### Method 1: Using relative paths (recommended)

When you mount a local directory to `/data`, you can use simple relative paths for your input and output files:

```bash
# Mount current directory and use relative paths
docker run -v $(pwd):/data sbom-technical-lag -in your-sbom-file.json -out results.json
```

#### Method 2: Using absolute paths

You can also use absolute paths that include the `/data` prefix:

```bash
# Mount current directory and use absolute paths
docker run -v $(pwd):/data sbom-technical-lag -in /data/your-sbom-file.json -out /data/results.json
```

### Examples

```bash
# Analyze an example SBOM in the current directory
docker run -v $(pwd):/data sbom-technical-lag -in sbom-go.json -out results.json

# Analyze an example SBOM from a subdirectory
docker run -v $(pwd):/data sbom-technical-lag -in examples/sbom-go.json -out results.json

# Print results to console (omit the -out parameter)
docker run -v $(pwd):/data sbom-technical-lag -in examples/sbom-go.json
```

## Output

Complete example outputs are available in the [examples](examples) directory.
Generally, the results are calculated for the whole project and then separated for the different types of package
scopes (direct, transitive, optional).
