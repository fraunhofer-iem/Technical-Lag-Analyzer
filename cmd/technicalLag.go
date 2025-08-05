package main

import (
	"encoding/json"
	"flag"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"log"
	"log/slog"
	"os"
	"sbom-technical-lag/internal/sbom"
	"sbom-technical-lag/internal/technicalLag"
	"time"
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

type TechLagStats struct {
	Libdays                        float64       `json:"libdays"`
	MissedReleases                 int64         `json:"missedReleases"`
	NumComponents                  int           `json:"numComponents"`
	HighestLibdays                 float64       `json:"highestLibdays"`
	HighestMissedReleases          int64         `json:"highestMissedReleases"`
	ComponentHighestMissedReleases cdx.Component `json:"componentHighestMissedReleases,omitempty"`
	ComponentHighestLibdays        cdx.Component `json:"componentHighestLibdays,omitempty"`
}

type Result struct {
	Opt        TechLagStats `json:"optional"`
	Prod       TechLagStats `json:"production"`
	DirectOpt  TechLagStats `json:"directOptional"`
	DirectProd TechLagStats `json:"directProduction"`
	Timestamp  int64        `json:"timestamp"`
}

// updateTechLagStats updates the TechLagStats fields with the given technical lag information
func updateTechLagStats(stats *TechLagStats, libdays float64, missedReleases int64, c cdx.Component) {
	stats.Libdays += libdays
	stats.MissedReleases += missedReleases
	stats.NumComponents++
	if missedReleases > stats.HighestMissedReleases {
		stats.HighestMissedReleases = missedReleases
		stats.ComponentHighestMissedReleases = c
	}
	if libdays > stats.HighestLibdays {
		stats.HighestLibdays = libdays
		stats.ComponentHighestLibdays = c
	}
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

	// Initialize Result struct to track all statistics
	result := Result{
		Opt:        TechLagStats{},
		Prod:       TechLagStats{},
		DirectOpt:  TechLagStats{},
		DirectProd: TechLagStats{},
	}

	for k, v := range cm {
		if k.Scope == "" || k.Scope == "required" {
			updateTechLagStats(&result.Prod, v.Libdays, v.VersionDistance.MissedReleases, k)
		} else {
			updateTechLagStats(&result.Opt, v.Libdays, v.VersionDistance.MissedReleases, k)
		}
	}

	directDeps, err := sbom.GetDirectDeps(bom)
	if err != nil {
		logger.Warn("Failed to get direct dependencies", "err", err)
	}

	for _, dep := range directDeps {
		tl := cm[dep]
		if dep.Scope == "" || dep.Scope == "required" {
			updateTechLagStats(&result.DirectProd, tl.Libdays, tl.VersionDistance.MissedReleases, dep)
		} else {
			updateTechLagStats(&result.DirectOpt, tl.Libdays, tl.VersionDistance.MissedReleases, dep)
		}
	}

	result.Timestamp = time.Now().Unix()

	logger.Info("Number components", "prod", result.Prod.NumComponents, "opt", result.Opt.NumComponents)
	logger.Info("Libdays", "prod", result.Prod.Libdays, "opt", result.Opt.Libdays)
	logger.Info("Missed releases", "prod", result.Prod.MissedReleases, "opt", result.Opt.MissedReleases)

	logger.Info("Number direct components", "prod", result.DirectProd.NumComponents, "opt", result.DirectOpt.NumComponents)
	logger.Info("Libdays direct", "prod", result.DirectProd.Libdays, "opt", result.DirectOpt.Libdays)
	logger.Info("Missed releases direct", "prod", result.DirectProd.MissedReleases, "opt", result.DirectOpt.MissedReleases)

	// Store results in a file if the out path is provided
	if *out != "" {
		resultFile, err := os.Create(*out)
		if err != nil {
			logger.Error("Failed to create output file", "err", err)
		} else {
			defer resultFile.Close()

			// Marshal the result to JSON
			jsonData, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				logger.Error("Failed to marshal result to JSON", "err", err)
				return
			}

			// Write JSON data to the file
			_, err = resultFile.Write(jsonData)
			if err != nil {
				logger.Error("Failed to write JSON data to file", "err", err)
				return
			}

			logger.Info("Results written to file in JSON format", "path", *out)
		}
	}

	elapsed := time.Since(start)
	logger.Info("Finished libyear calculation", "time elapsed", elapsed)
}
