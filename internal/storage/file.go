package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileStorage implements Storage using JSON files in a directory hierarchy.
//
// Reports are stored in a directory structure like:
//
//	{basePath}/history/{project-hash}/{timestamp}_{id-prefix}.json
//
// Where:
//   - basePath: Root storage directory (e.g., ~/.cra or ./.cra)
//   - project-hash: SHA256 hash of project path (first 16 chars)
//   - timestamp: YYYY-MM-DD_HH-MM-SS format
//   - id-prefix: First 6 chars of report UUID
//
// This structure provides:
//   - Easy manual inspection (human-readable filenames)
//   - Project isolation (each project in its own directory)
//   - Chronological ordering (timestamp in filename)
//   - Uniqueness (UUID prefix prevents collisions)
//
// Example directory structure:
//
//	~/.cra/history/
//	  a1b2c3d4e5f6g7h8/
//	    2025-12-20_14-30-45_abc123.json
//	    2025-12-20_15-22-10_def456.json
//	  9f8e7d6c5b4a3210/
//	    2025-12-19_10-15-30_ghi789.json
type FileStorage struct {
	basePath string // Root directory for storage
}

// NewFileStorage creates a new file-based storage backend.
//
// The basePath parameter specifies where reports will be stored. The directory
// will be created if it doesn't exist. A typical setup uses:
//   - ~/.cra for user-level storage
//   - ./.cra for project-level storage
//
// Returns an error if the directory cannot be created or accessed.
//
// Example:
//
//	storage, err := NewFileStorage(filepath.Join(homeDir, ".cra"))
//	if err != nil {
//	    return fmt.Errorf("failed to create storage: %w", err)
//	}
//	defer storage.Close()
func NewFileStorage(basePath string) (*FileStorage, error) {
	// Expand home directory if present
	if strings.HasPrefix(basePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		basePath = filepath.Join(homeDir, basePath[2:])
	}

	// Create base directory if it doesn't exist
	historyDir := filepath.Join(basePath, "history")
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create history directory: %w", err)
	}

	return &FileStorage{
		basePath: basePath,
	}, nil
}

// Save stores a report as a JSON file.
//
// The report is written to a file named with the timestamp and ID prefix.
// If a file with the same name exists, it will be overwritten.
//
// The write is atomic: the file is first written to a temporary location
// and then renamed to the final path to prevent partial writes.
func (fs *FileStorage) Save(ctx context.Context, report *StoredReport) error {
	// Get project directory
	projectDir, err := fs.getProjectDir(report.ProjectPath)
	if err != nil {
		return err
	}

	// Create project directory if it doesn't exist
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Generate filename
	filename := fs.generateFilename(report.Timestamp, report.ID)
	filePath := filepath.Join(projectDir, filename)

	// Marshal report to JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	// Write atomically (write to temp file, then rename)
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath) // Clean up temp file on failure
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// GetLatest retrieves the most recent report for a project.
//
// Reports are sorted by filename (which includes timestamp) to find the latest.
// Returns nil if no reports exist for the project.
func (fs *FileStorage) GetLatest(ctx context.Context, projectPath string) (*StoredReport, error) {
	projectDir, err := fs.getProjectDir(projectPath)
	if err != nil {
		return nil, err
	}

	// Check if project directory exists
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return nil, nil // No reports for this project
	}

	// List all report files
	files, err := filepath.Glob(filepath.Join(projectDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list reports: %w", err)
	}

	if len(files) == 0 {
		return nil, nil // No reports
	}

	// Sort files by name (timestamp is in filename, so this gives chronological order)
	sort.Strings(files)

	// Get the last (most recent) file
	latestFile := files[len(files)-1]

	return fs.loadReport(latestFile)
}

// GetByID retrieves a specific report by ID.
//
// Searches all project directories for a report with a matching ID.
// This is slower than GetLatest as it may need to search multiple directories.
func (fs *FileStorage) GetByID(ctx context.Context, id string) (*StoredReport, error) {
	historyDir := filepath.Join(fs.basePath, "history")

	// List all project directories
	entries, err := os.ReadDir(historyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("report not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read history directory: %w", err)
	}

	// Search each project directory for the report
	idPrefix := id
	if len(id) > 6 {
		idPrefix = id[:6]
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectDir := filepath.Join(historyDir, entry.Name())
		files, err := filepath.Glob(filepath.Join(projectDir, fmt.Sprintf("*_%s.json", idPrefix)))
		if err != nil {
			continue
		}

		// Check each matching file
		for _, file := range files {
			report, err := fs.loadReport(file)
			if err != nil {
				continue
			}

			if report.ID == id {
				return report, nil
			}
		}
	}

	return nil, fmt.Errorf("report not found: %s", id)
}

// List returns metadata for reports matching the criteria.
//
// Reports are filtered by project path and ListOptions, then sorted by
// timestamp (newest first). Only metadata is returned for efficiency.
func (fs *FileStorage) List(ctx context.Context, projectPath string, opts ListOptions) ([]*ReportMetadata, error) {
	projectDir, err := fs.getProjectDir(projectPath)
	if err != nil {
		return nil, err
	}

	// Check if project directory exists
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return []*ReportMetadata{}, nil // No reports
	}

	// Load all reports from directory
	reports, err := fs.loadReportsFromDirectory(projectDir)
	if err != nil {
		return nil, err
	}

	// Filter and extract metadata
	metadata := fs.filterAndExtractMetadata(reports, opts)

	// Sort by timestamp (newest first)
	fs.sortByTimestamp(metadata)

	// Apply pagination
	return fs.applyPagination(metadata, opts), nil
}

// loadReportsFromDirectory loads all JSON report files from a directory
func (fs *FileStorage) loadReportsFromDirectory(projectDir string) ([]*StoredReport, error) {
	files, err := filepath.Glob(filepath.Join(projectDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list reports: %w", err)
	}

	var reports []*StoredReport
	for _, file := range files {
		report, err := fs.loadReport(file)
		if err != nil {
			continue // Skip files that can't be loaded
		}
		reports = append(reports, report)
	}

	return reports, nil
}

// filterAndExtractMetadata filters reports by time and extracts metadata
func (fs *FileStorage) filterAndExtractMetadata(reports []*StoredReport, opts ListOptions) []*ReportMetadata {
	var metadata []*ReportMetadata

	for _, report := range reports {
		if !fs.passesTimeFilters(report, opts) {
			continue
		}

		meta := fs.extractMetadata(report)
		metadata = append(metadata, meta)
	}

	return metadata
}

// passesTimeFilters checks if a report passes the time filter criteria
func (fs *FileStorage) passesTimeFilters(report *StoredReport, opts ListOptions) bool {
	if !opts.Since.IsZero() && report.Timestamp.Before(opts.Since) {
		return false
	}
	if !opts.Until.IsZero() && report.Timestamp.After(opts.Until) {
		return false
	}
	return true
}

// extractMetadata extracts metadata from a stored report
func (fs *FileStorage) extractMetadata(report *StoredReport) *ReportMetadata {
	meta := &ReportMetadata{
		ID:          report.ID,
		ProjectPath: report.ProjectPath,
		Timestamp:   report.Timestamp,
		GitCommit:   report.GitCommit,
		GitBranch:   report.GitBranch,
	}

	if report.Result != nil {
		meta.TotalFiles = report.Result.TotalFiles
		meta.TotalLines = report.Result.TotalLines
		meta.IssueCount = len(report.Result.Issues)

		if report.Result.Metrics != nil {
			meta.AvgComplexity = report.Result.Metrics.AverageComplexity
		}
		if report.Result.Coverage != nil {
			meta.AvgCoverage = report.Result.Coverage.AverageCoverage
		}
	}

	return meta
}

// sortByTimestamp sorts metadata by timestamp (newest first)
func (fs *FileStorage) sortByTimestamp(metadata []*ReportMetadata) {
	sort.Slice(metadata, func(i, j int) bool {
		return metadata[i].Timestamp.After(metadata[j].Timestamp)
	})
}

// applyPagination applies offset and limit to metadata
func (fs *FileStorage) applyPagination(metadata []*ReportMetadata, opts ListOptions) []*ReportMetadata {
	start := opts.Offset
	if start > len(metadata) {
		start = len(metadata)
	}

	end := len(metadata)
	if opts.Limit > 0 && start+opts.Limit < end {
		end = start + opts.Limit
	}

	return metadata[start:end]
}

// Delete removes a report by ID.
//
// Searches for and deletes the file containing the report.
// Returns an error if the report doesn't exist.
func (fs *FileStorage) Delete(ctx context.Context, id string) error {
	// Find the report file
	report, err := fs.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Construct file path
	projectDir, err := fs.getProjectDir(report.ProjectPath)
	if err != nil {
		return err
	}

	filename := fs.generateFilename(report.Timestamp, report.ID)
	filePath := filepath.Join(projectDir, filename)

	// Delete the file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete report: %w", err)
	}

	return nil
}

// Close cleans up resources (no-op for file storage).
func (fs *FileStorage) Close() error {
	return nil
}

// getProjectDir returns the storage directory for a project.
//
// The directory name is a hash of the project path to avoid filesystem
// issues with special characters while keeping projects isolated.
func (fs *FileStorage) getProjectDir(projectPath string) (string, error) {
	// Make path absolute for consistent hashing
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Hash the path
	hash := sha256.Sum256([]byte(absPath))
	hashStr := hex.EncodeToString(hash[:])[:16] // Use first 16 chars

	return filepath.Join(fs.basePath, "history", hashStr), nil
}

// generateFilename creates a filename from timestamp and ID.
//
// Format: YYYY-MM-DD_HH-MM-SS_{id-prefix}.json
// Example: 2025-12-20_14-30-45_abc123.json
func (fs *FileStorage) generateFilename(timestamp time.Time, id string) string {
	timeStr := timestamp.Format("2006-01-02_15-04-05")
	idPrefix := id
	if len(id) > 6 {
		idPrefix = id[:6]
	}
	return fmt.Sprintf("%s_%s.json", timeStr, idPrefix)
}

// loadReport loads a report from a JSON file.
func (fs *FileStorage) loadReport(filePath string) (*StoredReport, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read report file: %w", err)
	}

	var report StoredReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal report: %w", err)
	}

	return &report, nil
}
