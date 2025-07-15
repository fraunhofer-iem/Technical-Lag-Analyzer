package main

import (
	"errors"
	"flag"
	"log"
	"log/slog"
	"os"
	"sbom-technical-lag/internal/deps"
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

// validates output path and sets default tmpdir
// as default value if *p == ""
// p - path
func ValidateOutPath(p *string) error {
	// set default value if needed
	if *p == "" {
		dir := os.TempDir()
		*p = dir
	}

	f, err := os.Stat(*p)
	if err != nil {
		return err
	}

	if !f.IsDir() {
		return errors.New("out path must be a directory")
	}

	return nil
}

// validates input path and sets a current working dir
// as default value if *p == ""
// p - path
func ValidateInPath(p *string) (os.FileInfo, error) {

	// set default value if needed
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
	// get input path and check for correctness
	flag.Parse()

	logger := SetUpLogging(*logLevel)

	_, err := ValidateInPath(in)
	if err != nil {
		log.Fatal(err)
	}

	err = ValidateOutPath(out)
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
		//Exit
		log.Fatal("No components in sbom")
	}

	componentToVersions := make(map[cdx.Component][]deps.VersionsApiResponse)
	errC := 0

	for _, c := range *bom.Components {

		depsRes, err := deps.GetVersions(c.PackageURL)
		if err != nil {
			logger.Warn("Deps.dev api query failed", "purl", c.PackageURL, "err", err)
			errC++
			continue
		}
		componentToVersions[c] = depsRes.Versions
	}

	if errC > 0 {
		logger.Warn("Requests failed", "counter", errC)
	}

	elapsed := time.Since(start)
	logger.Info("Finished libyear calculation", "time elapsed", elapsed)
}
