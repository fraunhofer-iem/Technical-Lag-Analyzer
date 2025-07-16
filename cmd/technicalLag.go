package main

import (
	"flag"
	"log"
	"log/slog"
	"os"
	"sbom-technical-lag/internal/sbom"
	"sbom-technical-lag/internal/technicalLag"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

var in = flag.String("in", "", "Path to SBOM")
var out = flag.String("out", "", "File to write the SBOM to")
var logLevel = flag.Int("logLevel", 0, "Can be 0 for INFO, -4 for DEBUG, 4 for WARN, or 8 for ERROR. Defaults to INFO.")

// SetUpLogging sets up the logging for the application based on the log level provided
// logLevel: int - the log level to set the logger to, defaults to error
// returns: *slog.Logger - the logger to be used for logging
// sets the logger as slog.Default
func SetUpLogging(logLevel int) *slog.Logger {

	var lvl slog.Level

	switch {
	case logLevel < int(slog.LevelInfo):
		lvl = slog.LevelDebug
	case logLevel < int(slog.LevelWarn):
		lvl = slog.LevelInfo
	case logLevel < int(slog.LevelError):
		lvl = slog.LevelWarn
	default:
		lvl = slog.LevelError
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))

	slog.SetDefault(logger)
	return logger
}

// ValidateInPath validates an input path and sets a current working dir
// as default value if *p == ""
// p - path
func ValidateInPath(p *string) (os.FileInfo, error) {

	// set the default value if needed
	if *p == "" {
		dir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		*p = dir
	}
	f, err := os.Stat(*p)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func main() {

	start := time.Now()
	// get an input path and check for correctness
	flag.Parse()

	logger := SetUpLogging(*logLevel)

	_, err := ValidateInPath(in)
	if err != nil {
		log.Fatal(err)
	}

	logger.Info("Starting libyear calculation", "path", *in, "out", *out)

	file, err := os.Open(*in)

	// Decode the BOM
	bom := new(cdx.BOM)
	decoder := cdx.NewBOMDecoder(file, cdx.BOMFileFormatJSON)
	if err = decoder.Decode(bom); err != nil {
		panic(err)
	}

	if bom.Components == nil {
		log.Fatal("No components in sbom")
	}

	cm, err := technicalLag.Calculate(bom)
	if err != nil {
		log.Fatal(err)
	}

	var libdaysOpt float64
	var libdaysProd float64
	var missedReleasesProd int64
	var missedReleasesOpt int64
	numComponentsProd := 0
	numComponentsOpt := 0

	for k, v := range cm {
		if k.Scope == "" || k.Scope == "required" {
			libdaysProd += v.Libdays
			missedReleasesProd += v.VersionDistance.MissedReleases
			numComponentsProd++
		} else {
			libdaysOpt += v.Libdays
			missedReleasesOpt += v.VersionDistance.MissedReleases
			numComponentsOpt++
		}
	}

	directDeps, err := sbom.GetDirectDeps(bom)
	if err != nil {
		logger.Warn("Failed to get direct dependencies", "err", err)
	}

	var libdaysDirectOpt float64
	var libdaysDirectProd float64
	var missedReleasesDirectProd int64
	var missedReleasesDirectOpt int64
	numDirectComponentsProd := 0
	numDirectComponentsOpt := 0
	for _, dep := range directDeps {
		tl := cm[dep]
		if dep.Scope == "" || dep.Scope == "required" {
			libdaysDirectProd += tl.Libdays
			missedReleasesDirectProd += tl.VersionDistance.MissedReleases
			numDirectComponentsProd++
		} else {
			libdaysDirectOpt += tl.Libdays
			missedReleasesDirectOpt += tl.VersionDistance.MissedReleases
			numDirectComponentsOpt++
		}
	}

	logger.Info("Number components", "prod", numComponentsProd, "opt", numComponentsOpt)
	logger.Info("Libdays", "prod", libdaysProd, "opt", libdaysOpt)
	logger.Info("Missed releases", "prod", missedReleasesProd, "opt", missedReleasesOpt)

	logger.Info("Number direct components", "prod", numDirectComponentsProd, "opt", numDirectComponentsOpt)
	logger.Info("Libdays direct", "prod", libdaysDirectProd, "opt", libdaysDirectOpt)
	logger.Info("Missed releases direct", "prod", missedReleasesDirectProd, "opt", missedReleasesDirectOpt)

	elapsed := time.Since(start)
	logger.Info("Finished libyear calculation", "time elapsed", elapsed)
}
