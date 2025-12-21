package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// SQLiteStorage implements Storage using a SQLite database.
//
// Reports are stored in a relational database with the following schema:
//   - reports table: Full report data with denormalized metrics
//   - Indexes on project_path and timestamp for efficient queries
//
// This backend provides:
//   - Efficient querying and filtering
//   - Structured storage with integrity constraints
//   - Better performance for large numbers of reports
//   - ACID guarantees for concurrent access
//
// The database uses WAL (Write-Ahead Logging) mode for better concurrency
// and prepared statements for performance.
//
// Example usage:
//   storage, err := NewSQLiteStorage("/path/to/history.db")
//   if err != nil {
//       return err
//   }
//   defer storage.Close()
type SQLiteStorage struct {
	db *sql.DB
}

// Database schema
const schema = `
CREATE TABLE IF NOT EXISTS reports (
    id TEXT PRIMARY KEY,
    project_path TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    git_commit TEXT,
    git_branch TEXT,
    result_json TEXT NOT NULL,

    -- Denormalized metrics for fast queries
    total_files INTEGER,
    total_lines INTEGER,
    total_functions INTEGER,
    avg_complexity REAL,
    avg_coverage REAL,
    issue_count INTEGER,

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_project_timestamp
    ON reports(project_path, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_timestamp
    ON reports(timestamp DESC);
`

// NewSQLiteStorage creates a new SQLite-based storage backend.
//
// The dbPath parameter specifies the database file location. The file and
// any necessary parent directories will be created if they don't exist.
//
// The database is initialized with:
//   - Schema creation (reports table and indexes)
//   - WAL mode enabled for better concurrency
//   - Foreign keys enabled
//
// Returns an error if the database cannot be opened or initialized.
//
// Example:
//   storage, err := NewSQLiteStorage(filepath.Join(homeDir, ".cra", "history.db"))
//   if err != nil {
//       return fmt.Errorf("failed to create storage: %w", err)
//   }
//   defer storage.Close()
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	// Expand home directory if present
	if strings.HasPrefix(dbPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(homeDir, dbPath[2:])
	}

	// Create parent directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create schema
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &SQLiteStorage{
		db: db,
	}, nil
}

// Save stores a report in the database.
//
// The report is inserted with both the full JSON data and denormalized
// metrics for efficient querying. If a report with the same ID exists,
// it will be replaced.
func (ss *SQLiteStorage) Save(ctx context.Context, report *StoredReport) error {
	// Marshal result to JSON
	resultJSON, err := json.Marshal(report.Result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	// Extract denormalized metrics
	var totalFiles, totalLines, totalFunctions, issueCount int
	var avgComplexity, avgCoverage float64

	if report.Result != nil {
		totalFiles = report.Result.TotalFiles
		totalLines = report.Result.TotalLines
		totalFunctions = report.Result.TotalFunctions
		issueCount = len(report.Result.Issues)

		if report.Result.Metrics != nil {
			avgComplexity = report.Result.Metrics.AverageComplexity
		}

		if report.Result.Coverage != nil {
			avgCoverage = report.Result.Coverage.AverageCoverage
		}
	}

	// Insert or replace report
	query := `
		INSERT OR REPLACE INTO reports (
			id, project_path, timestamp, git_commit, git_branch, result_json,
			total_files, total_lines, total_functions, avg_complexity, avg_coverage, issue_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = ss.db.ExecContext(ctx, query,
		report.ID,
		report.ProjectPath,
		report.Timestamp,
		report.GitCommit,
		report.GitBranch,
		string(resultJSON),
		totalFiles,
		totalLines,
		totalFunctions,
		avgComplexity,
		avgCoverage,
		issueCount,
	)

	if err != nil {
		return fmt.Errorf("failed to save report: %w", err)
	}

	return nil
}

// GetLatest retrieves the most recent report for a project.
//
// Uses an indexed query on project_path and timestamp for efficiency.
// Returns nil if no reports exist for the project.
func (ss *SQLiteStorage) GetLatest(ctx context.Context, projectPath string) (*StoredReport, error) {
	// Make path absolute for consistent matching
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	query := `
		SELECT id, project_path, timestamp, git_commit, git_branch, result_json
		FROM reports
		WHERE project_path = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var report StoredReport
	var resultJSON string

	err = ss.db.QueryRowContext(ctx, query, absPath).Scan(
		&report.ID,
		&report.ProjectPath,
		&report.Timestamp,
		&report.GitCommit,
		&report.GitBranch,
		&resultJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No reports for this project
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query latest report: %w", err)
	}

	// Unmarshal result JSON
	if err := json.Unmarshal([]byte(resultJSON), &report.Result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &report, nil
}

// GetByID retrieves a specific report by ID.
//
// Uses a primary key lookup for fast retrieval.
func (ss *SQLiteStorage) GetByID(ctx context.Context, id string) (*StoredReport, error) {
	query := `
		SELECT id, project_path, timestamp, git_commit, git_branch, result_json
		FROM reports
		WHERE id = ?
	`

	var report StoredReport
	var resultJSON string

	err := ss.db.QueryRowContext(ctx, query, id).Scan(
		&report.ID,
		&report.ProjectPath,
		&report.Timestamp,
		&report.GitCommit,
		&report.GitBranch,
		&resultJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("report not found: %s", id)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query report: %w", err)
	}

	// Unmarshal result JSON
	if err := json.Unmarshal([]byte(resultJSON), &report.Result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &report, nil
}

// List returns metadata for reports matching the criteria.
//
// Efficiently queries using denormalized metrics without loading full JSON.
// Supports filtering by date range and pagination.
func (ss *SQLiteStorage) List(ctx context.Context, projectPath string, opts ListOptions) ([]*ReportMetadata, error) {
	// Make path absolute for consistent matching
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Build query with filters
	query := `
		SELECT id, project_path, timestamp, git_commit, git_branch,
		       total_files, total_lines, avg_complexity, avg_coverage, issue_count
		FROM reports
		WHERE project_path = ?
	`
	args := []interface{}{absPath}

	// Add time filters
	if !opts.Since.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, opts.Since)
	}
	if !opts.Until.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, opts.Until)
	}

	// Order by timestamp descending
	query += " ORDER BY timestamp DESC"

	// Add pagination
	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	// Execute query
	rows, err := ss.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query reports: %w", err)
	}
	defer rows.Close()

	// Collect results
	var metadata []*ReportMetadata
	for rows.Next() {
		var meta ReportMetadata
		err := rows.Scan(
			&meta.ID,
			&meta.ProjectPath,
			&meta.Timestamp,
			&meta.GitCommit,
			&meta.GitBranch,
			&meta.TotalFiles,
			&meta.TotalLines,
			&meta.AvgComplexity,
			&meta.AvgCoverage,
			&meta.IssueCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan report metadata: %w", err)
		}

		metadata = append(metadata, &meta)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reports: %w", err)
	}

	return metadata, nil
}

// Delete removes a report by ID.
//
// Returns an error if the report doesn't exist.
func (ss *SQLiteStorage) Delete(ctx context.Context, id string) error {
	result, err := ss.db.ExecContext(ctx, "DELETE FROM reports WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete report: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("report not found: %s", id)
	}

	return nil
}

// Close closes the database connection.
//
// Should be called when the storage is no longer needed to release
// database resources.
func (ss *SQLiteStorage) Close() error {
	if ss.db != nil {
		return ss.db.Close()
	}
	return nil
}
