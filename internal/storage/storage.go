package storage

import (
	"context"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
)

// Storage defines the interface for persisting and retrieving analysis reports.
//
// Implementations provide different storage backends (file-based, database, etc.)
// while maintaining a consistent API for saving and loading reports. All
// operations accept a context for cancellation and timeout support.
//
// The storage layer enables historical tracking and comparison features by
// persisting analysis results with metadata like timestamps and Git information.
type Storage interface {
	// Save stores an analysis report with metadata.
	//
	// The report is assigned a unique ID and stored with the provided metadata
	// (timestamp, Git commit, branch). Returns an error if storage fails.
	//
	// Example:
	//   report := &StoredReport{
	//       ID: "abc-123",
	//       ProjectPath: "/path/to/project",
	//       Timestamp: time.Now(),
	//       GitCommit: "def456",
	//       Result: analysisResult,
	//   }
	//   err := storage.Save(ctx, report)
	Save(ctx context.Context, report *StoredReport) error

	// GetLatest retrieves the most recent report for a project.
	//
	// Returns the latest report based on timestamp, or nil if no reports
	// exist for the project. The projectPath should be an absolute path
	// for consistent matching across runs.
	//
	// Example:
	//   latest, err := storage.GetLatest(ctx, "/path/to/project")
	//   if latest != nil {
	//       // Compare with current analysis
	//   }
	GetLatest(ctx context.Context, projectPath string) (*StoredReport, error)

	// GetByID retrieves a specific report by its unique ID.
	//
	// Returns the report with the given ID, or an error if not found.
	// Useful for retrieving specific historical reports for comparison.
	//
	// Example:
	//   report, err := storage.GetByID(ctx, "abc-123")
	GetByID(ctx context.Context, id string) (*StoredReport, error)

	// List returns reports for a project, optionally filtered and paginated.
	//
	// Returns metadata for matching reports ordered by timestamp (newest first).
	// Use ListOptions to filter by date range and implement pagination.
	//
	// Example:
	//   opts := ListOptions{Limit: 10, Since: oneWeekAgo}
	//   reports, err := storage.List(ctx, "/path/to/project", opts)
	List(ctx context.Context, projectPath string, opts ListOptions) ([]*ReportMetadata, error)

	// Delete removes a report by ID.
	//
	// Returns an error if the report doesn't exist or deletion fails.
	// Use with caution as this operation is irreversible.
	//
	// Example:
	//   err := storage.Delete(ctx, "abc-123")
	Delete(ctx context.Context, id string) error

	// Close cleans up storage resources.
	//
	// Should be called when the storage is no longer needed to release
	// any open connections, file handles, or other resources.
	//
	// Example:
	//   defer storage.Close()
	Close() error
}

// StoredReport represents a saved analysis report with metadata.
//
// This type wraps an AnalysisResult with additional metadata needed for
// historical tracking and comparison:
//   - Unique ID for retrieval
//   - Project path for grouping reports
//   - Timestamp for chronological ordering
//   - Git metadata for correlation with code changes
//
// StoredReport is the primary type passed to Storage.Save() and returned
// from Storage.Get*() methods.
type StoredReport struct {
	// Unique identifier for this report (UUID)
	ID string `json:"id"`

	// Absolute path to the analyzed project
	ProjectPath string `json:"project_path"`

	// When this analysis was performed
	Timestamp time.Time `json:"timestamp"`

	// Git commit hash at time of analysis (optional)
	GitCommit string `json:"git_commit,omitempty"`

	// Git branch name at time of analysis (optional)
	GitBranch string `json:"git_branch,omitempty"`

	// The complete analysis result
	Result *analyzer.AnalysisResult `json:"result"`
}

// ReportMetadata contains summary information about a stored report.
//
// This type provides lightweight report information for listing operations,
// without loading the full AnalysisResult. Useful for displaying report
// history or selecting reports for comparison.
//
// Returned by Storage.List() for efficient browsing of report history.
type ReportMetadata struct {
	// Unique identifier for this report
	ID string `json:"id"`

	// Absolute path to the analyzed project
	ProjectPath string `json:"project_path"`

	// When this analysis was performed
	Timestamp time.Time `json:"timestamp"`

	// Git commit hash at time of analysis (optional)
	GitCommit string `json:"git_commit,omitempty"`

	// Git branch name at time of analysis (optional)
	GitBranch string `json:"git_branch,omitempty"`

	// Summary metrics (optional, for quick display)
	TotalFiles    int     `json:"total_files,omitempty"`
	TotalLines    int     `json:"total_lines,omitempty"`
	AvgComplexity float64 `json:"avg_complexity,omitempty"`
	AvgCoverage   float64 `json:"avg_coverage,omitempty"`
	IssueCount    int     `json:"issue_count,omitempty"`
}

// ListOptions specifies filtering and pagination for Storage.List().
//
// Use these options to:
//   - Limit the number of results (pagination)
//   - Skip results (offset-based pagination)
//   - Filter by date range (since/until)
//
// Example:
//
//	// Get last 10 reports from the past week
//	opts := ListOptions{
//	    Limit: 10,
//	    Since: time.Now().AddDate(0, 0, -7),
//	}
type ListOptions struct {
	// Maximum number of results to return (0 = unlimited)
	Limit int

	// Number of results to skip (for pagination)
	Offset int

	// Only include reports after this time (inclusive)
	Since time.Time

	// Only include reports before this time (inclusive)
	Until time.Time
}
