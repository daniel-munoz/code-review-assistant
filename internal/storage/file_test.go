package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/daniel-munoz/code-review-assistant/internal/analyzer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileStorage(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, storage)
	defer storage.Close()

	// Verify directory was created
	historyDir := filepath.Join(tmpDir, "history")
	assert.DirExists(t, historyDir)
}

func TestFileStorage_SaveAndGetByID(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)
	defer storage.Close()

	// Create a test report
	report := &StoredReport{
		ID:          uuid.New().String(),
		ProjectPath: "/test/project",
		Timestamp:   time.Now(),
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
}

func TestFileStorage_GetLatest(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectPath := "/test/project"

	// Save multiple reports with different timestamps
	report1 := createTestReport(projectPath, time.Now().Add(-2*time.Hour))
	report2 := createTestReport(projectPath, time.Now().Add(-1*time.Hour))
	report3 := createTestReport(projectPath, time.Now())

	require.NoError(t, storage.Save(ctx, report1))
	require.NoError(t, storage.Save(ctx, report2))
	require.NoError(t, storage.Save(ctx, report3))

	// Get latest should return report3
	latest, err := storage.GetLatest(ctx, projectPath)
	require.NoError(t, err)
	require.NotNil(t, latest)

	assert.Equal(t, report3.ID, latest.ID)
}

func TestFileStorage_GetLatest_NoReports(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Get latest for project with no reports
	latest, err := storage.GetLatest(ctx, "/nonexistent/project")
	require.NoError(t, err)
	assert.Nil(t, latest)
}

func TestFileStorage_List(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectPath := "/test/project"

	// Save 5 reports
	now := time.Now()
	for i := 0; i < 5; i++ {
		report := createTestReport(projectPath, now.Add(-time.Duration(i)*time.Hour))
		require.NoError(t, storage.Save(ctx, report))
	}

	// List all reports
	metadata, err := storage.List(ctx, projectPath, ListOptions{})
	require.NoError(t, err)
	assert.Len(t, metadata, 5)

	// Verify sorted by timestamp (newest first)
	for i := 0; i < len(metadata)-1; i++ {
		assert.True(t, metadata[i].Timestamp.After(metadata[i+1].Timestamp))
	}
}

func TestFileStorage_List_WithLimit(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectPath := "/test/project"

	// Save 5 reports
	now := time.Now()
	for i := 0; i < 5; i++ {
		report := createTestReport(projectPath, now.Add(-time.Duration(i)*time.Hour))
		require.NoError(t, storage.Save(ctx, report))
	}

	// List with limit
	metadata, err := storage.List(ctx, projectPath, ListOptions{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, metadata, 3)
}

func TestFileStorage_List_WithOffsetAndLimit(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectPath := "/test/project"

	// Save 5 reports
	now := time.Now()
	for i := 0; i < 5; i++ {
		report := createTestReport(projectPath, now.Add(-time.Duration(i)*time.Hour))
		require.NoError(t, storage.Save(ctx, report))
	}

	// List with offset and limit (pagination)
	metadata, err := storage.List(ctx, projectPath, ListOptions{Offset: 2, Limit: 2})
	require.NoError(t, err)
	assert.Len(t, metadata, 2)
}

func TestFileStorage_List_WithTimeFilter(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectPath := "/test/project"

	now := time.Now()
	oneDayAgo := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	// Save reports at different times
	report1 := createTestReport(projectPath, twoDaysAgo)
	report2 := createTestReport(projectPath, oneDayAgo)
	report3 := createTestReport(projectPath, now)

	require.NoError(t, storage.Save(ctx, report1))
	require.NoError(t, storage.Save(ctx, report2))
	require.NoError(t, storage.Save(ctx, report3))

	// List reports since one day ago
	metadata, err := storage.List(ctx, projectPath, ListOptions{Since: oneDayAgo.Add(-time.Second)})
	require.NoError(t, err)
	assert.Len(t, metadata, 2) // Should get report2 and report3
}

func TestFileStorage_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Save a report
	report := createTestReport("/test/project", time.Now())
	require.NoError(t, storage.Save(ctx, report))

	// Verify it exists
	retrieved, err := storage.GetByID(ctx, report.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Delete the report
	err = storage.Delete(ctx, report.ID)
	require.NoError(t, err)

	// Verify it's deleted
	retrieved, err = storage.GetByID(ctx, report.ID)
	assert.Error(t, err)
	assert.Nil(t, retrieved)
}

func TestFileStorage_Delete_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Try to delete non-existent report
	err = storage.Delete(ctx, "nonexistent-id")
	assert.Error(t, err)
}

func TestFileStorage_MultipleProjects(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Save reports for different projects
	project1 := "/test/project1"
	project2 := "/test/project2"

	report1 := createTestReport(project1, time.Now())
	report2 := createTestReport(project2, time.Now())

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

func TestFileStorage_GenerateFilename(t *testing.T) {
	storage := &FileStorage{basePath: "/tmp"}

	timestamp := time.Date(2025, 12, 20, 14, 30, 45, 0, time.UTC)
	id := "abc123def456"

	filename := storage.generateFilename(timestamp, id)

	assert.Equal(t, "2025-12-20_14-30-45_abc123.json", filename)
}

func TestFileStorage_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Save a report
	report := createTestReport("/test/project", time.Now())
	require.NoError(t, storage.Save(ctx, report))

	// Verify no .tmp files left behind
	projectDir, err := storage.getProjectDir(report.ProjectPath)
	require.NoError(t, err)

	tmpFiles, err := filepath.Glob(filepath.Join(projectDir, "*.tmp"))
	require.NoError(t, err)
	assert.Empty(t, tmpFiles, "should not have temporary files after successful save")
}

func TestFileStorage_MetadataExtraction(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	projectPath := "/test/project"

	// Create a report with full metrics
	report := &StoredReport{
		ID:          uuid.New().String(),
		ProjectPath: projectPath,
		Timestamp:   time.Now(),
		GitCommit:   "abc123",
		GitBranch:   "main",
		Result: &analyzer.AnalysisResult{
			ProjectPath:    projectPath,
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

// Helper function to create test reports
func createTestReport(projectPath string, timestamp time.Time) *StoredReport {
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
