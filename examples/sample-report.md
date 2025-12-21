# Code Review Assistant - Analysis Report

**Project:** internal/comparison  
**Analyzed:** 2025-12-20 16:29:38  

---

## Summary

| Metric | Value |
|--------|-------|
| Total Files | 1 |
| Total Lines | 333 |
| Code Lines | 202 (60.7%) |
| Comment Lines | 88 (26.4%) |
| Blank Lines | 43 (12.9%) |
| Total Functions | 10 |

## Aggregate Metrics

| Metric | Value |
|--------|-------|
| Average Function Length | 19.6 lines |
| Function Length (95th %ile) | 61 lines |
| Comment Ratio | 26.4% |
| Average Complexity | 3.4 |
| Complexity (95th %ile) | 6 |

## Issues Found (11)

⚠️ **[WARNING]** Function exceeds length threshold
  - **File:** `internal/comparison/comparison.go:87`
  - **Function:** `*Comparator.calculateDeltas`
  - **Lines:** 61 (threshold: 50)

ℹ️ **[INFO]** Magic number should be replaced with a named constant: 5.0
  - **File:** `internal/comparison/comparison.go:37`
  - **Function:** `NewComparator`

ℹ️ **[INFO]** Magic number should be replaced with a named constant: 100
  - **File:** `internal/comparison/comparison.go:120`
  - **Function:** `*Comparator.calculateDeltas`

ℹ️ **[INFO]** Magic number should be replaced with a named constant: 100
  - **File:** `internal/comparison/comparison.go:121`
  - **Function:** `*Comparator.calculateDeltas`

ℹ️ **[INFO]** Magic number should be replaced with a named constant: 100
  - **File:** `internal/comparison/comparison.go:133`
  - **Function:** `*Comparator.calculateDeltas`

ℹ️ **[INFO]** Magic number should be replaced with a named constant: 100
  - **File:** `internal/comparison/comparison.go:134`
  - **Function:** `*Comparator.calculateDeltas`

ℹ️ **[INFO]** Function has 5 return statements
  - **File:** `internal/comparison/comparison.go:175`
  - **Function:** `*Comparator.detectTrend`
  - **Return Statements:** 5 (threshold: 3)

ℹ️ **[INFO]** Magic number should be replaced with a named constant: 100.0
  - **File:** `internal/comparison/comparison.go:248`
  - **Function:** `*Comparator.percentChange`

ℹ️ **[INFO]** Magic number should be replaced with a named constant: 100.0
  - **File:** `internal/comparison/comparison.go:250`
  - **Function:** `*Comparator.percentChange`

ℹ️ **[INFO]** Function has 4 return statements
  - **File:** `internal/comparison/comparison.go:307`
  - **Function:** `TrendDirection.String`
  - **Return Statements:** 4 (threshold: 3)

ℹ️ **[INFO]** Function has 4 return statements
  - **File:** `internal/comparison/comparison.go:321`
  - **Function:** `TrendDirection.Icon`
  - **Return Statements:** 4 (threshold: 3)

## Largest Files

| Rank | File | Lines |
|------|------|-------|
| 1 | `internal/comparison/comparison.go` | 333 |

## Most Complex Functions

| Rank | Function | Complexity | Lines | File |
|------|----------|------------|-------|------|
| 1 | `*Comparator.detectTrend` | 6 | 25 | `internal/comparison/comparison.go` |
| 2 | `*Comparator.categorizeIssues` | 6 | 30 | `internal/comparison/comparison.go` |
| 3 | `*Comparator.calculateDeltas` | 5 | 61 | `internal/comparison/comparison.go` |
| 4 | `TrendDirection.String` | 4 | 12 | `internal/comparison/comparison.go` |
| 5 | `TrendDirection.Icon` | 4 | 12 | `internal/comparison/comparison.go` |
| 6 | `*Comparator.percentChange` | 3 | 9 | `internal/comparison/comparison.go` |
| 7 | `NewComparator` | 2 | 8 | `internal/comparison/comparison.go` |
| 8 | `*Comparator.Compare` | 2 | 22 | `internal/comparison/comparison.go` |
| 9 | `*Comparator.detectTrends` | 1 | 14 | `internal/comparison/comparison.go` |
| 10 | `*Comparator.issueKey` | 1 | 3 | `internal/comparison/comparison.go` |

## Test Coverage

| Metric | Value |
|--------|-------|
| Average Coverage | 100.0% |
| Total Packages | 1 |

## Dependencies

| Metric | Value |
|--------|-------|
| Total Packages | 1 |

---

*Analysis complete.*
