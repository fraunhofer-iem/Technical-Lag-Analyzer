package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sbom-technical-lag/internal/technicalLag"
	"syscall"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

// Config holds the application configuration
type Config struct {
	InputPath  string
	OutputPath string
	LogLevel   int
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	config := parseFlags()

	logger := setupLogging(config.LogLevel)
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	start := time.Now()
	logger.Info("Starting technical lag calculation", "input", config.InputPath, "output", config.OutputPath)

	if err := validateInputPath(&config.InputPath); err != nil {
		return fmt.Errorf("invalid input path: %w", err)
	}

	bom, err := loadSBOM(config.InputPath)
	if err != nil {
		return fmt.Errorf("failed to load SBOM: %w", err)
	}

	if bom.Components == nil {
		return errors.New("no components found in SBOM")
	}

	componentMetrics, err := technicalLag.Calculate(ctx, bom)
	if err != nil {
		return fmt.Errorf("failed to calculate technical lag: %w", err)
	}

	result, err := technicalLag.CreateResult(bom, componentMetrics)
	if err != nil {
		return fmt.Errorf("failed to create result: %w", err)
	}

	logger.Info("Calculation completed", "details", result.String())

	if config.OutputPath != "" {
		if err := saveResults(result, config.OutputPath); err != nil {
			return fmt.Errorf("failed to save results: %w", err)
		}
		logger.Info("Results written to file", "path", config.OutputPath)
	}

	elapsed := time.Since(start)
	logger.Info("Technical lag calculation finished", "duration", elapsed)

	return nil
}

func parseFlags() Config {
	var config Config

	flag.StringVar(&config.InputPath, "in", "", "Path to SBOM file")
	flag.StringVar(&config.OutputPath, "out", "", "Output file for results (JSON format)")
	flag.IntVar(&config.LogLevel, "log-level", 0, "Log level: -4 (DEBUG), 0 (INFO), 4 (WARN), 8 (ERROR)")
	flag.Parse()

	return config
}

// setupLogging configures structured logging based on the provided log level
func setupLogging(logLevel int) *slog.Logger {
	var level slog.Level

	switch {
	case logLevel <= int(slog.LevelDebug):
		level = slog.LevelDebug
	case logLevel <= int(slog.LevelInfo):
		level = slog.LevelInfo
	case logLevel <= int(slog.LevelWarn):
		level = slog.LevelWarn
	default:
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	return slog.New(handler)
}

// validateInputPath validates the input path and sets current working directory as default
func validateInputPath(path *string) error {
	if *path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
		*path = cwd
	}

	if _, err := os.Stat(*path); err != nil {
		return fmt.Errorf("path does not exist or is not accessible: %w", err)
	}

	return nil
}

// loadSBOM loads and decodes a CycloneDX SBOM from the specified file path
func loadSBOM(filePath string) (*cdx.BOM, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SBOM file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			slog.Default().Warn("Failed to close SBOM file", "error", closeErr)
		}
	}()

	bom := new(cdx.BOM)
	decoder := cdx.NewBOMDecoder(file, cdx.BOMFileFormatJSON)
	if err := decoder.Decode(bom); err != nil {
		return nil, fmt.Errorf("failed to decode SBOM: %w", err)
	}

	return bom, nil
}

// saveResults saves the technical lag results to a JSON file
func saveResults(result technicalLag.Result, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			slog.Default().Warn("Failed to close output file", "error", closeErr)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to encode results to JSON: %w", err)
	}

	return nil
}
