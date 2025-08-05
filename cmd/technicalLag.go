package main

import (
	"encoding/json"
	"flag"
	"log"
	"log/slog"
	"os"
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
		log.Fatal(err)
	}

	if bom.Components == nil {
		log.Fatal("No components in sbom")
	}

	cm, err := technicalLag.Calculate(bom)
	if err != nil {
		log.Fatal(err)
	}

	result, err := technicalLag.CreateResult(bom, cm)
	if err != nil {
		log.Fatal(err)
	}
	logger.Info("Result", "details", result.String())

	// Initialize Result struct to track all statistics

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
