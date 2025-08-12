# HotPath Analysis

HotPath Analysis identifies components that contribute significantly (more than 50% by default) to your project's technical lag. This helps prioritize which dependencies to update first for maximum impact.

## What is a HotPath?

A hotpath is the minimal set of components that together contribute more than the specified threshold (default: 50%) to the total technical lag in a given scope. HotPaths help you identify:

- **Critical Dependencies**: Components that disproportionately contribute to technical lag
- **Update Priorities**: Which dependencies to update first for maximum impact
- **Risk Assessment**: Dependencies that pose the highest technical debt risk

## Metrics

HotPath analysis supports two different technical lag metrics:

### 1. Libyears (Time-based)
- Measures how far behind your dependencies are in terms of **time**
- Calculated as the time difference between your version's release date and the latest version's release date
- Expressed in **days** behind the latest version
- Better for understanding the age-based risk of dependencies

### 2. Version Distance (Release-based)
- Measures how far behind your dependencies are in terms of **number of releases**
- Counts the actual number of releases between your version and the latest version
- Expressed in **missed releases**
- Better for understanding update effort and potential breaking changes

## Scopes

HotPath analysis categorizes dependencies into different scopes:

- **Production**: Runtime dependencies required for production
- **Optional**: Development dependencies, test dependencies, optional dependencies
- **DirectProduction**: Direct production dependencies (your immediate dependencies)
- **DirectOptional**: Direct optional dependencies (dev dependencies you directly declared)

## Usage

### Command Line Tool

The dedicated `hotpaths.go` command provides focused hotpath analysis:

```bash
# Basic analysis - shows libyears hotpaths for all scopes
go run cmd/hotpaths.go -in sbom.json

# Analyze version distance instead of libyears
go run cmd/hotpaths.go -in sbom.json -metric versionDistance

# Analyze both metrics
go run cmd/hotpaths.go -in sbom.json -metric both

# Focus on specific scope
go run cmd/hotpaths.go -in sbom.json -scope production

# Change threshold to 70%
go run cmd/hotpaths.go -in sbom.json -threshold 70

# Output formats
go run cmd/hotpaths.go -in sbom.json -format json -out hotpaths.json
go run cmd/hotpaths.go -in sbom.json -format csv -out hotpaths.csv
```

### Options

- `-in`: Path to SBOM file (required)
- `-out`: Output file path (optional, defaults to stdout)
- `-metric`: Metric to analyze (`libyears`, `versionDistance`, or `both`)
- `-scope`: Scope to analyze (`all`, `production`, `optional`, `direct`, `directProduction`, `directOptional`)
- `-format`: Output format (`text`, `json`, `csv`)
- `-threshold`: Threshold percentage for hotpath (default: 50.0)
- `-log-level`: Logging level (-4=DEBUG, 0=INFO, 4=WARN, 8=ERROR)

### Integrated Analysis

HotPath analysis is also included in the main technical lag analysis:

```bash
go run cmd/technicalLag.go -in sbom.json -out results.json
```

The results include a `hotPaths` section with complete hotpath analysis.

## Example Output

### Text Format

```
=== HotPath Analysis ===
Threshold: 50.0%

Most Critical Component: production (libyears) (71.3% of scope)
Most Fragmented: optional (versionDistance)

=== Libyears HotPaths ===

--- Production ---
Total Lag: 267.16 days
HotPath Coverage: 71.3% (1 components)

HotPath Components:
1. @kurkle/color@0.3.4
   Contribution: 190.50 days (71.3% of total, 71.3% cumulative)
   PURL: pkg:npm/%40kurkle/color@0.3.4

Top Non-HotPath Contributors:
2. @jridgewell/sourcemap-codec@1.5.2: 43.07 days (16.1%)
3. @babel/types@7.27.7: 27.82 days (10.4%)
```

### CSV Format

```csv
scope,metric,component_name,component_version,purl,contribution,percentage_of_total,cumulative_percent,is_hotpath
production,libyears,@kurkle/color,0.3.4,pkg:npm/%40kurkle/color@0.3.4,190.50,71.30,71.30,true
production,libyears,@jridgewell/sourcemap-codec,1.5.2,pkg:npm/%40jridgewell/sourcemap-codec@1.5.2,43.07,16.12,0.00,false
```

## Interpreting Results

### HotPath Coverage
- **High Coverage (>70%)**: Few components cause most lag - focus updates here
- **Medium Coverage (50-70%)**: Moderate concentration - several key updates needed
- **Low Coverage (<50%)**: Fragmented lag - many small updates needed

### Component Contribution
- **Percentage of Total**: How much this component contributes to total lag in its scope
- **Cumulative Percent**: Running total when components are ordered by contribution
- **Is HotPath**: Whether this component is part of the minimal hotpath set

### Practical Recommendations

1. **Start with HotPath Components**: Update these first for maximum impact
2. **Consider Scope**: Direct dependencies are easier to update than transitive ones
3. **Balance Metrics**: 
   - High libyears = security/stability risk
   - High version distance = update effort/breaking changes
4. **Update Strategy**:
   - Single hotpath component with >80% contribution = immediate priority
   - Multiple hotpath components = coordinate updates
   - No clear hotpath = gradual across-the-board updates

## Examples by Project Type

### Well-Maintained Project
```
HotPath Coverage: 0% (0 components)
```
- No components significantly behind
- Suggests good maintenance practices

### Legacy Project
```
Most Critical: entities@4.5.0 (36.2% of scope)
HotPath Coverage: 71.8% (2 components)
```
- Clear update targets identified
- Focus on top 2 components for major impact

### Fragmented Lag
```
HotPath Coverage: 53.6% (8 components)
```
- Many components contribute to lag
- Consider broader update strategy
- May indicate need for dependency audit

## Integration with CI/CD

HotPath analysis can be integrated into your CI/CD pipeline to:

1. **Gate Deployments**: Fail builds if hotpath components exceed thresholds
2. **Update Planning**: Generate update priority lists automatically  
3. **Security Monitoring**: Track high-lag components for security implications
4. **Technical Debt Metrics**: Monitor hotpath trends over time

Example CI integration:
```bash
# Generate hotpath report
go run cmd/hotpaths.go -in sbom.json -format json -out hotpaths.json

# Check if any component contributes >80% of lag
if jq -r '.libyearsHotPaths[].hotPathComponents[]? | select(.percentageOfTotal > 80)' hotpaths.json | head -1; then
  echo "Critical hotpath component detected - update required"
  exit 1
fi
```

## API Integration

The hotpath analysis functionality is available programmatically:

```go
// Create hotpath analysis
analysis := technicalLag.CreateHotPathAnalysis(result)

// Access libyears hotpaths
for _, hotPath := range analysis.LibyearsHotPaths {
    fmt.Printf("Scope: %s, Coverage: %.1f%%\n", 
        hotPath.Scope, hotPath.HotPathCoverage)
    
    for _, comp := range hotPath.HotPathComponents {
        fmt.Printf("  %s@%s: %.1f%%\n", 
            comp.Component.Name, 
            comp.Component.Version,
            comp.PercentageOfTotal)
    }
}
```
