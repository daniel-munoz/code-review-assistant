package storage

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSQLiteStorage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	assert.NotNil(t, storage)
	defer storage.Close()

	// Verify database file was created
	assert.FileExists(t, dbPath)
}

func TestNewSQLiteStorage_InMemory(t *testing.T) {
	// Use in-memory database for testing
	storage, err := NewSQLiteStorage(":memory:")
	require.NoError(t, err)
	assert.NotNil(t, storage)
	defer storage.Close()
}

func TestSQLiteStorage_SaveAndGetByID(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	require.NoError(t, err)
	defer storage.Close()

	// Create a test report
	report := &StoredReport{
		ID:          uuid.New().String(),
		ProjectPath: "/test/project",
		Timestamp:   time.Now().Truncate(time.Second), // SQLite has second precision
		GitCommit:   "abc123",
		GitBranch:   "main",
		Result: &analyzer.AnalysisResult{
			ProjectPath:    "/test/project",
			TotalFiles:     10,
			TotalLines:     1000,
			TotalCodeLines: 700,
			TotalFunctions: 50,
			Metrics: &analyzer.AggregateMetrics{
				AverageComplexity: 3.5,
			},
		},
	}

	// Save the report
	ctx := context.Background()
	err = storage.Save(ctx, report)
	require.NoError(t, err)

	// Retrieve by ID
	retrieved, err := storage.GetByID(ctx, report.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Verify fields
	assert.Equal(t, report.ID, retrieved.ID)
	assert.Equal(t, report.ProjectPath, retrieved.ProjectPath)
	assert.Equal(t, report.GitCommit, retrieved.GitCommit)
	assert.Equal(t, report.GitBranch, retrieved.GitBranch)
	assert.Equal(t, report.Result.TotalFiles, retrieved.Result.TotalFiles)
	assert.Equal(t, report.Result.TotalLines, retrieved.Result.TotalLines)
	assert.WithinDuration(t, report.Timestamp, retrieved.Timestamp, time.Second)
}

func TestSQLiteStorage_SaveReplace(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	id := uuid.New().String()

	// Save initial report
	report1 := &StoredReport{
		ID:          id,
		ProjectPath: "/test/project",
		Timestamp:   time.Now().Truncate(time.Second),
		Result: &analyzer.AnalysisResult{
			TotalFiles: 10,
		},
	}
	require.NoError(t, storage.Save(ctx, report1))

	// Save again with same ID (should replace)
	report2 := &StoredReport{
		ID:          id,
		ProjectPath: "/test/project",
		Timestamp:   time.Now().Truncate(time.Second),
		Result: &analyzer.AnalysisResult{
			TotalFiles: 20,
		},
	}
	require.NoError(t, storage.Save(ctx, report2))

	// Retrieve should get the updated version
	retrieved, err := storage.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, 20, retrieved.Result.TotalFiles)
}

func TestSQLiteStorage_GetLatest(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectPath := "/test/project"

	// Make path absolute for consistent matching
	absPath, err := filepath.Abs(projectPath)
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)

	// Save multiple reports with different timestamps
	report1 := createTestReportSQL(absPath, now.Add(-2*time.Hour))
	report2 := createTestReportSQL(absPath, now.Add(-1*time.Hour))
	report3 := createTestReportSQL(absPath, now)

	require.NoError(t, storage.Save(ctx, report1))
	require.NoError(t, storage.Save(ctx, report2))
	require.NoError(t, storage.Save(ctx, report3))

	// Get latest should return report3
	latest, err := storage.GetLatest(ctx, projectPath)
	require.NoError(t, err)
	require.NotNil(t, latest)

	assert.Equal(t, report3.ID, latest.ID)
}

func TestSQLiteStorage_GetLatest_NoReports(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Get latest for project with no reports
	latest, err := storage.GetLatest(ctx, "/nonexistent/project")
	require.NoError(t, err)
	assert.Nil(t, latest)
}

func TestSQLiteStorage_List(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectPath := "/test/project"
	absPath, err := filepath.Abs(projectPath)
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)

	// Save 5 reports
	for i := 0; i < 5; i++ {
		report := createTestReportSQL(absPath, now.Add(-time.Duration(i)*time.Hour))
		require.NoError(t, storage.Save(ctx, report))
	}

	// List all reports
	metadata, err := storage.List(ctx, projectPath, ListOptions{})
	require.NoError(t, err)
	assert.Len(t, metadata, 5)

	// Verify sorted by timestamp (newest first)
	for i := 0; i < len(metadata)-1; i++ {
		assert.True(t, metadata[i].Timestamp.After(metadata[i+1].Timestamp) ||
			metadata[i].Timestamp.Equal(metadata[i+1].Timestamp))
	}
}

func TestSQLiteStorage_List_WithLimit(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectPath := "/test/project"
	absPath, err := filepath.Abs(projectPath)
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)

	// Save 5 reports
	for i := 0; i < 5; i++ {
		report := createTestReportSQL(absPath, now.Add(-time.Duration(i)*time.Hour))
		require.NoError(t, storage.Save(ctx, report))
	}

	// List with limit
	metadata, err := storage.List(ctx, projectPath, ListOptions{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, metadata, 3)
}

func TestSQLiteStorage_List_WithOffsetAndLimit(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectPath := "/test/project"
	absPath, err := filepath.Abs(projectPath)
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)

	// Save 5 reports
	for i := 0; i < 5; i++ {
		report := createTestReportSQL(absPath, now.Add(-time.Duration(i)*time.Hour))
		require.NoError(t, storage.Save(ctx, report))
	}

	// List with offset and limit
	metadata, err := storage.List(ctx, projectPath, ListOptions{Offset: 2, Limit: 2})
	require.NoError(t, err)
	assert.Len(t, metadata, 2)
}

func TestSQLiteStorage_List_WithTimeFilter(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectPath := "/test/project"
	absPath, err := filepath.Abs(projectPath)
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)
	oneDayAgo := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	// Save reports at different times
	report1 := createTestReportSQL(absPath, twoDaysAgo)
	report2 := createTestReportSQL(absPath, oneDayAgo)
	report3 := createTestReportSQL(absPath, now)

	require.NoError(t, storage.Save(ctx, report1))
	require.NoError(t, storage.Save(ctx, report2))
	require.NoError(t, storage.Save(ctx, report3))

	// List reports since one day ago
	metadata, err := storage.List(ctx, projectPath, ListOptions{Since: oneDayAgo})
	require.NoError(t, err)
	assert.Len(t, metadata, 2) // Should get report2 and report3
}

func TestSQLiteStorage_Delete(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Save a report
	report := createTestReportSQL("/test/project", time.Now().Truncate(time.Second))
	require.NoError(t, storage.Save(ctx, report))

	// Verify it exists
	retrieved, err := storage.GetByID(ctx, report.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Delete the report
	err = storage.Delete(ctx, report.ID)
	require.NoError(t, err)

	// Verify it's deleted
	_, err = storage.GetByID(ctx, report.ID)
	assert.Error(t, err)
}

func TestSQLiteStorage_Delete_NotFound(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Try to delete non-existent report
	err = storage.Delete(ctx, "nonexistent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSQLiteStorage_MetadataExtraction(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectPath := "/test/project"
	absPath, err := filepath.Abs(projectPath)
	require.NoError(t, err)

	// Create a report with full metrics
	report := &StoredReport{
		ID:          uuid.New().String(),
		ProjectPath: absPath,
		Timestamp:   time.Now().Truncate(time.Second),
		GitCommit:   "abc123",
		GitBranch:   "main",
		Result: &analyzer.AnalysisResult{
			ProjectPath:    absPath,
			TotalFiles:     42,
			TotalLines:     5000,
			TotalCodeLines: 3500,
			TotalFunctions: 100,
			Metrics: &analyzer.AggregateMetrics{
				AverageComplexity: 4.5,
			},
			Issues: []*analyzer.Issue{
				{Severity: "warning", Message: "test"},
				{Severity: "info", Message: "test2"},
			},
			Coverage: &analyzer.CoverageReport{
				AverageCoverage: 75.5,
			},
		},
	}

	require.NoError(t, storage.Save(ctx, report))

	// List and verify metadata extraction
	metadata, err := storage.List(ctx, projectPath, ListOptions{})
	require.NoError(t, err)
	require.Len(t, metadata, 1)

	meta := metadata[0]
	assert.Equal(t, report.ID, meta.ID)
	assert.Equal(t, 42, meta.TotalFiles)
	assert.Equal(t, 5000, meta.TotalLines)
	assert.Equal(t, 4.5, meta.AvgComplexity)
	assert.Equal(t, 75.5, meta.AvgCoverage)
	assert.Equal(t, 2, meta.IssueCount)
}

func TestSQLiteStorage_MultipleProjects(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	project1 := "/test/project1"
	project2 := "/test/project2"

	absPath1, err := filepath.Abs(project1)
	require.NoError(t, err)
	absPath2, err := filepath.Abs(project2)
	require.NoError(t, err)

	// Save reports for different projects
	report1 := createTestReportSQL(absPath1, time.Now().Truncate(time.Second))
	report2 := createTestReportSQL(absPath2, time.Now().Truncate(time.Second))

	require.NoError(t, storage.Save(ctx, report1))
	require.NoError(t, storage.Save(ctx, report2))

	// Get latest for project1
	latest1, err := storage.GetLatest(ctx, project1)
	require.NoError(t, err)
	assert.Equal(t, report1.ID, latest1.ID)

	// Get latest for project2
	latest2, err := storage.GetLatest(ctx, project2)
	require.NoError(t, err)
	assert.Equal(t, report2.ID, latest2.ID)
}

func TestSQLiteStorage_Close(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	require.NoError(t, err)

	// Close should not error
	err = storage.Close()
	assert.NoError(t, err)

	// Calling Close again should not panic
	err = storage.Close()
	assert.NoError(t, err)
}

func TestSQLiteStorage_WALMode(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	// Verify WAL mode is enabled
	var journalMode string
	err = storage.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	require.NoError(t, err)
	assert.Equal(t, "wal", strings.ToLower(journalMode))
}

// Helper function to create test reports for SQL storage
func createTestReportSQL(projectPath string, timestamp time.Time) *StoredReport {
	return &StoredReport{
		ID:          uuid.New().String(),
		ProjectPath: projectPath,
		Timestamp:   timestamp,
		GitCommit:   "abc123",
		GitBranch:   "main",
		Result: &analyzer.AnalysisResult{
			ProjectPath:    projectPath,
			TotalFiles:     10,
			TotalLines:     1000,
			TotalCodeLines: 700,
			TotalFunctions: 50,
			Metrics: &analyzer.AggregateMetrics{
				AverageComplexity: 3.5,
			},
		},
	}
}
