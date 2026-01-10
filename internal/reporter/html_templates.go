package reporter

// htmlTemplate is the base HTML template for the dashboard
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Code Review Dashboard - {{.ProjectName}}</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        :root {
            --primary: #3b82f6;
            --success: #10b981;
            --warning: #f59e0b;
            --danger: #ef4444;
            --bg: #f9fafb;
            --card-bg: white;
            --text: #111827;
            --text-secondary: #6b7280;
            --border: #e5e7eb;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background-color: var(--bg);
            color: var(--text);
            line-height: 1.6;
            padding: 2rem;
        }

        .dashboard {
            max-width: 1400px;
            margin: 0 auto;
        }

        header {
            background: linear-gradient(135deg, var(--primary), #2563eb);
            color: white;
            padding: 2rem;
            border-radius: 0.5rem;
            margin-bottom: 2rem;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }

        header h1 {
            font-size: 2rem;
            margin-bottom: 0.5rem;
        }

        .meta {
            display: flex;
            gap: 2rem;
            font-size: 0.9rem;
            opacity: 0.9;
            flex-wrap: wrap;
        }

        .meta-item {
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .section {
            background: var(--card-bg);
            border-radius: 0.5rem;
            padding: 1.5rem;
            margin-bottom: 1.5rem;
            box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
        }

        .section-title {
            font-size: 1.5rem;
            margin-bottom: 1rem;
            color: var(--text);
            border-bottom: 2px solid var(--border);
            padding-bottom: 0.5rem;
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-bottom: 1.5rem;
        }

        .stat-card {
            background: var(--bg);
            padding: 1rem;
            border-radius: 0.375rem;
            border-left: 4px solid var(--primary);
        }

        .stat-label {
            font-size: 0.875rem;
            color: var(--text-secondary);
            margin-bottom: 0.25rem;
        }

        .stat-value {
            font-size: 1.5rem;
            font-weight: 600;
            color: var(--text);
        }

        .stat-secondary {
            font-size: 0.875rem;
            color: var(--text-secondary);
            margin-top: 0.25rem;
        }

        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 1rem;
        }

        th {
            background: var(--bg);
            padding: 0.75rem;
            text-align: left;
            font-weight: 600;
            border-bottom: 2px solid var(--border);
        }

        td {
            padding: 0.75rem;
            border-bottom: 1px solid var(--border);
        }

        tr:hover {
            background: var(--bg);
        }

        .issue-item {
            padding: 1rem;
            margin-bottom: 0.75rem;
            border-left: 4px solid var(--border);
            background: var(--bg);
            border-radius: 0.25rem;
        }

        .issue-header {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            margin-bottom: 0.5rem;
        }

        .issue-message {
            font-weight: 500;
            color: var(--text);
        }

        .issue-details {
            font-size: 0.875rem;
            color: var(--text-secondary);
            margin-left: 1.5rem;
        }

        .issue-detail-item {
            margin-top: 0.25rem;
        }

        .severity-error {
            border-left-color: var(--danger);
        }

        .severity-warning {
            border-left-color: var(--warning);
        }

        .severity-info {
            border-left-color: var(--primary);
        }

        .badge {
            display: inline-block;
            padding: 0.25rem 0.5rem;
            border-radius: 0.25rem;
            font-size: 0.75rem;
            font-weight: 600;
            text-transform: uppercase;
        }

        .badge-error {
            background: #fee2e2;
            color: #991b1b;
        }

        .badge-warning {
            background: #fef3c7;
            color: #92400e;
        }

        .badge-info {
            background: #dbeafe;
            color: #1e40af;
        }

        .badge-success {
            background: #d1fae5;
            color: #065f46;
        }

        .comparison-section {
            background: #f0f9ff;
            border: 1px solid #bfdbfe;
            border-radius: 0.5rem;
            padding: 1.5rem;
            margin-bottom: 1.5rem;
        }

        .trend-indicator {
            display: inline-flex;
            align-items: center;
            gap: 0.25rem;
            font-weight: 500;
        }

        .trend-improving {
            color: var(--success);
        }

        .trend-worsening {
            color: var(--danger);
        }

        .trend-stable {
            color: var(--text-secondary);
        }

        code {
            background: var(--bg);
            padding: 0.125rem 0.375rem;
            border-radius: 0.25rem;
            font-family: 'Courier New', monospace;
            font-size: 0.875rem;
        }

        .charts-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
            gap: 1.5rem;
            margin-top: 1rem;
        }

        .chart-container {
            background: var(--bg);
            padding: 1.5rem;
            border-radius: 0.5rem;
            min-height: 300px;
        }

        .chart-title {
            font-size: 1.1rem;
            margin-bottom: 1rem;
            color: var(--text);
        }

        .heatmap-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
            gap: 0.5rem;
            padding: 1rem;
            background: var(--bg);
            border-radius: 0.5rem;
        }

        .heatmap-cell {
            aspect-ratio: 1;
            border-radius: 0.375rem;
            padding: 0.5rem;
            display: flex;
            flex-direction: column;
            justify-content: space-between;
            align-items: flex-start;
            cursor: pointer;
            transition: transform 0.2s, box-shadow 0.2s;
            position: relative;
            overflow: hidden;
            color: white;
            text-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
        }

        .heatmap-cell:hover {
            transform: scale(1.05);
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
            z-index: 10;
        }

        .heatmap-filename {
            font-size: 0.75rem;
            font-weight: 600;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            max-width: 100%;
        }

        .heatmap-complexity {
            font-size: 0.875rem;
            font-weight: 700;
            align-self: flex-end;
        }

        /* Size variants for heatmap cells */
        .heatmap-cell.size-1 {
            grid-column: span 1;
            grid-row: span 1;
        }

        .heatmap-cell.size-2 {
            grid-column: span 1;
            grid-row: span 1;
        }

        .heatmap-cell.size-3 {
            grid-column: span 2;
            grid-row: span 1;
        }

        .heatmap-cell.size-4 {
            grid-column: span 2;
            grid-row: span 2;
        }

        .heatmap-cell.size-5 {
            grid-column: span 3;
            grid-row: span 2;
        }

        .empty-state {
            text-align: center;
            padding: 3rem;
            color: var(--text-secondary);
        }

        .empty-state-icon {
            font-size: 3rem;
            margin-bottom: 1rem;
        }

        /* Collapsible sections */
        details {
            border: 1px solid var(--border);
            border-radius: 0.5rem;
            padding: 1rem;
            background: var(--bg);
        }

        details summary {
            cursor: pointer;
            font-weight: 600;
            user-select: none;
            padding: 0.5rem;
            border-radius: 0.375rem;
            transition: background-color 0.2s;
        }

        details summary:hover {
            background: var(--card-bg);
        }

        details[open] summary {
            margin-bottom: 1rem;
            border-bottom: 1px solid var(--border);
        }

        /* Add arrow indicator */
        details summary::marker {
            content: '▶ ';
        }

        details[open] summary::marker {
            content: '▼ ';
        }

        footer {
            text-align: center;
            padding: 2rem;
            color: var(--text-secondary);
            font-size: 0.875rem;
        }

        @media (max-width: 768px) {
            body {
                padding: 1rem;
            }

            header h1 {
                font-size: 1.5rem;
            }

            .stats-grid {
                grid-template-columns: 1fr;
            }

            .charts-grid {
                grid-template-columns: 1fr;
            }

            .heatmap-grid {
                grid-template-columns: repeat(auto-fill, minmax(60px, 1fr));
                gap: 0.375rem;
            }

            .heatmap-cell {
                padding: 0.375rem;
            }

            .heatmap-filename {
                font-size: 0.65rem;
            }

            .heatmap-complexity {
                font-size: 0.75rem;
            }

            /* Simplify size variants on mobile */
            .heatmap-cell.size-3,
            .heatmap-cell.size-4,
            .heatmap-cell.size-5 {
                grid-column: span 2;
                grid-row: span 1;
            }

            table {
                font-size: 0.875rem;
            }

            th, td {
                padding: 0.5rem;
            }
        }

        @media print {
            body {
                background: white;
                padding: 0;
            }

            .section {
                box-shadow: none;
                page-break-inside: avoid;
            }

            header {
                background: var(--primary);
            }
        }
    </style>
</head>
<body>
    <div class="dashboard">
        <header>
            <h1>📊 Code Review Dashboard</h1>
            <div class="meta">
                <div class="meta-item">
                    <span>📁 Project:</span>
                    <strong>{{.ProjectPath}}</strong>
                </div>
                <div class="meta-item">
                    <span>🕐 Analyzed:</span>
                    <strong>{{.Timestamp}}</strong>
                </div>
            </div>
        </header>

        {{if .Comparison}}
        <div class="comparison-section">
            <h2 class="section-title">📊 Comparison with Previous Report</h2>
            <p><strong>Previous Report:</strong> {{.Comparison.PreviousTimestamp.Format "2006-01-02 15:04:05"}}</p>

            <div style="margin-top: 1rem;">
                <h3 style="font-size: 1.1rem; margin-bottom: 0.5rem;">Trends</h3>
                <div style="display: flex; gap: 2rem; flex-wrap: wrap;">
                    <div class="trend-indicator {{trendClass .Comparison.Trends.Complexity}}">
                        {{trendIcon .Comparison.Trends.Complexity}} Complexity: {{.Comparison.Trends.Complexity.String}}
                    </div>
                    <div class="trend-indicator {{trendClass .Comparison.Trends.Coverage}}">
                        {{trendIcon .Comparison.Trends.Coverage}} Coverage: {{.Comparison.Trends.Coverage.String}}
                    </div>
                    <div class="trend-indicator {{trendClass .Comparison.Trends.IssueCount}}">
                        {{trendIcon .Comparison.Trends.IssueCount}} Issues: {{.Comparison.Trends.IssueCount.String}}
                    </div>
                </div>
            </div>

            <table style="margin-top: 1rem;">
                <thead>
                    <tr>
                        <th>Metric</th>
                        <th>Previous</th>
                        <th>Current</th>
                        <th>Change</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td>Files</td>
                        <td>{{formatNumber .Comparison.Deltas.TotalFiles.Previous}}</td>
                        <td>{{formatNumber .Comparison.Deltas.TotalFiles.Current}}</td>
                        <td>{{formatDelta .Comparison.Deltas.TotalFiles.Change .Comparison.Deltas.TotalFiles.Percent}}</td>
                    </tr>
                    <tr>
                        <td>Lines</td>
                        <td>{{formatNumber .Comparison.Deltas.TotalLines.Previous}}</td>
                        <td>{{formatNumber .Comparison.Deltas.TotalLines.Current}}</td>
                        <td>{{formatDelta .Comparison.Deltas.TotalLines.Change .Comparison.Deltas.TotalLines.Percent}}</td>
                    </tr>
                    <tr>
                        <td>Functions</td>
                        <td>{{formatNumber .Comparison.Deltas.TotalFunctions.Previous}}</td>
                        <td>{{formatNumber .Comparison.Deltas.TotalFunctions.Current}}</td>
                        <td>{{formatDelta .Comparison.Deltas.TotalFunctions.Change .Comparison.Deltas.TotalFunctions.Percent}}</td>
                    </tr>
                    <tr>
                        <td>Avg Complexity</td>
                        <td>{{formatFloat .Comparison.Deltas.AvgComplexity.Previous}}</td>
                        <td>{{formatFloat .Comparison.Deltas.AvgComplexity.Current}}</td>
                        <td>{{printf "%+.2f (%+.1f%%)" .Comparison.Deltas.AvgComplexity.Change .Comparison.Deltas.AvgComplexity.Percent}}</td>
                    </tr>
                    <tr>
                        <td>Avg Coverage</td>
                        <td>{{formatFloat .Comparison.Deltas.AvgCoverage.Previous}}%</td>
                        <td>{{formatFloat .Comparison.Deltas.AvgCoverage.Current}}%</td>
                        <td>{{printf "%+.2f (%+.1f%%)" .Comparison.Deltas.AvgCoverage.Change .Comparison.Deltas.AvgCoverage.Percent}}</td>
                    </tr>
                    <tr>
                        <td>Issues</td>
                        <td>{{formatNumber .Comparison.Deltas.IssueCount.Previous}}</td>
                        <td>{{formatNumber .Comparison.Deltas.IssueCount.Current}}</td>
                        <td>{{formatDelta .Comparison.Deltas.IssueCount.Change .Comparison.Deltas.IssueCount.Percent}}</td>
                    </tr>
                </tbody>
            </table>
        </div>
        {{end}}

        <div class="section">
            <h2 class="section-title">📈 Summary Metrics</h2>
            <div class="stats-grid">
                <div class="stat-card">
                    <div class="stat-label">Total Files</div>
                    <div class="stat-value">{{formatNumber .TotalFiles}}</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Total Lines</div>
                    <div class="stat-value">{{formatNumber .TotalLines}}</div>
                    <div class="stat-secondary">
                        Code: {{formatNumber .TotalCodeLines}} ({{formatFloat .CodePercent}}%)
                    </div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Comment Lines</div>
                    <div class="stat-value">{{formatNumber .CommentLines}}</div>
                    <div class="stat-secondary">{{formatFloat .CommentPercent}}% of total</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Total Functions</div>
                    <div class="stat-value">{{formatNumber .TotalFunctions}}</div>
                </div>
            </div>
        </div>

        {{if .Metrics}}
        <div class="section">
            <h2 class="section-title">📊 Aggregate Metrics</h2>
            <div class="stats-grid">
                <div class="stat-card">
                    <div class="stat-label">Avg Function Length</div>
                    <div class="stat-value">{{formatFloat .Metrics.AverageFunctionLength}}</div>
                    <div class="stat-secondary">95th percentile: {{.Metrics.FunctionLengthP95}} lines</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Avg Complexity</div>
                    <div class="stat-value">{{formatFloat .Metrics.AverageComplexity}}</div>
                    <div class="stat-secondary">95th percentile: {{.Metrics.ComplexityP95}}</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Comment Ratio</div>
                    <div class="stat-value">{{formatFloat .Metrics.CommentRatio}}%</div>
                </div>
            </div>
        </div>
        {{end}}

        {{if .Coverage}}
        <div class="section">
            <h2 class="section-title">🧪 Test Coverage</h2>
            <div class="stats-grid">
                <div class="stat-card">
                    <div class="stat-label">Average Coverage</div>
                    <div class="stat-value">{{formatFloat .Coverage.AverageCoverage}}%</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Total Packages</div>
                    <div class="stat-value">{{len .Coverage.Packages}}</div>
                </div>
                {{if gt .Coverage.LowCoverageCount 0}}
                <div class="stat-card">
                    <div class="stat-label">Below Threshold</div>
                    <div class="stat-value" style="color: var(--warning);">{{.Coverage.LowCoverageCount}}</div>
                </div>
                {{end}}
            </div>

            {{if .Verbose}}
            <table>
                <thead>
                    <tr>
                        <th>Package</th>
                        <th>Coverage</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Coverage.Packages}}
                    <tr>
                        <td><code>{{.PackagePath}}</code></td>
                        <td>
                            {{if .Error}}
                                <span style="color: var(--danger);">Error: {{.Error}}</span>
                            {{else if .Skipped}}
                                <span style="color: var(--text-secondary);">No tests</span>
                            {{else}}
                                {{formatFloat .Coverage}}%
                            {{end}}
                        </td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
            {{end}}
        </div>
        {{end}}

        {{if .Dependencies}}
        <div class="section">
            <h2 class="section-title">📦 Dependencies</h2>
            <div class="stats-grid">
                <div class="stat-card">
                    <div class="stat-label">Total Packages</div>
                    <div class="stat-value">{{.Dependencies.TotalPackages}}</div>
                </div>
                {{if gt .Dependencies.HighImportCount 0}}
                <div class="stat-card">
                    <div class="stat-label">High Import Count</div>
                    <div class="stat-value" style="color: var(--warning);">{{.Dependencies.HighImportCount}}</div>
                </div>
                {{end}}
                {{if gt .Dependencies.HighExternalCount 0}}
                <div class="stat-card">
                    <div class="stat-label">High External Deps</div>
                    <div class="stat-value" style="color: var(--warning);">{{.Dependencies.HighExternalCount}}</div>
                </div>
                {{end}}
                {{if gt (len .Dependencies.CircularDependencies) 0}}
                <div class="stat-card">
                    <div class="stat-label">Circular Dependencies</div>
                    <div class="stat-value" style="color: var(--danger);">{{len .Dependencies.CircularDependencies}}</div>
                </div>
                {{end}}
            </div>

            {{if gt (len .Dependencies.CircularDependencies) 0}}
            <div style="margin-top: 1.5rem; padding: 1rem; background: #fee2e2; border-left: 4px solid var(--danger); border-radius: 0.25rem;">
                <h3 style="color: var(--danger); margin-bottom: 0.5rem;">⚠️ Circular Dependencies Detected</h3>
                <ul style="margin-left: 1.5rem;">
                    {{range $index, $cd := .Dependencies.CircularDependencies}}
                    <li style="margin-top: 0.5rem;">
                        <code>{{index $cd.Cycle 0}}</code>
                        {{range $i, $pkg := slice $cd.Cycle 1}}
                        → <code>{{$pkg}}</code>
                        {{end}}
                    </li>
                    {{end}}
                </ul>
            </div>
            {{end}}

            {{if .Verbose}}
            <details style="margin-top: 1.5rem;">
                <summary style="cursor: pointer; font-weight: 600;">Package Dependency Details</summary>
                <div style="margin-top: 1rem;">
                    {{range .Dependencies.Packages}}
                    <div style="margin-bottom: 1rem;">
                        <strong>{{.PackageName}}:</strong>
                        <div style="margin-left: 1rem; color: var(--text-secondary); font-size: 0.9rem;">
                            Total Imports: {{.TotalImports}} (Stdlib: {{len .StdlibImports}}, Internal: {{len .InternalImports}}, External: {{len .ExternalImports}})
                        </div>
                    </div>
                    {{end}}
                </div>
            </details>
            {{end}}
        </div>
        {{end}}

        {{if gt .IssueCount 0}}
        <div class="section">
            <h2 class="section-title">🔍 Issues Found ({{.IssueCount}})</h2>
            <div class="stats-grid" style="margin-bottom: 1.5rem;">
                <div class="stat-card">
                    <div class="stat-label">Errors</div>
                    <div class="stat-value" style="color: var(--danger);">{{.ErrorCount}}</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Warnings</div>
                    <div class="stat-value" style="color: var(--warning);">{{.WarningCount}}</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Info</div>
                    <div class="stat-value" style="color: var(--primary);">{{.InfoCount}}</div>
                </div>
            </div>

            <details style="margin-top: 1rem;">
                <summary>View All Issues</summary>
                <div style="margin-top: 1rem;">
                    {{range .Issues}}
                    <div class="issue-item {{severityClass .Severity}}">
                        <div class="issue-header">
                            <span>{{severityIcon .Severity}}</span>
                            <span class="badge badge-{{.Severity}}">{{.Severity}}</span>
                            <span class="issue-message">{{.Message}}</span>
                        </div>
                        <div class="issue-details">
                            {{if .File}}
                            <div class="issue-detail-item">
                                <strong>File:</strong> <code>{{.File}}{{if gt .Line 0}}:{{.Line}}{{end}}</code>
                            </div>
                            {{end}}
                            {{if .Function}}
                            <div class="issue-detail-item">
                                <strong>Function:</strong> <code>{{.Function}}</code>
                            </div>
                            {{end}}
                            {{if ne .Type "magic_number"}}
                            {{if eq .Type "duplicate_error_handling"}}
                            <div class="issue-detail-item">
                                <strong>Error checks:</strong> {{.Value}} (threshold: {{.Threshold}})
                            </div>
                            {{else if eq .Type "low_comment_ratio"}}
                            <div class="issue-detail-item">
                                <strong>Current:</strong> {{.Value}}% (recommended: >{{.Threshold}}%)
                            </div>
                            {{else if issueTypeLabel .Type}}
                            <div class="issue-detail-item">
                                <strong>{{issueTypeLabel .Type}}:</strong> {{.Value}} (threshold: {{.Threshold}})
                            </div>
                            {{end}}
                            {{end}}
                        </div>
                    </div>
                    {{end}}
                </div>
            </details>
        </div>
        {{else}}
        <div class="section">
            <div class="empty-state">
                <div class="empty-state-icon">✅</div>
                <h3>No Issues Found</h3>
                <p>Great job! No code quality issues were detected.</p>
            </div>
        </div>
        {{end}}

        {{if .ChartData}}
        <div class="section">
            <h2 class="section-title">📊 Visualizations</h2>

            <div class="charts-grid">
                {{if .ChartData.ComplexityDist}}
                <div class="chart-container">
                    <h3 class="chart-title">Complexity Distribution</h3>
                    <canvas id="complexityChart"></canvas>
                </div>
                {{end}}

                {{if .ChartData.CoverageBreakdown}}
                <div class="chart-container">
                    <h3 class="chart-title">Coverage by Package</h3>
                    <canvas id="coverageChart"></canvas>
                </div>
                {{end}}

                {{if .ChartData.IssueCounts}}
                <div class="chart-container">
                    <h3 class="chart-title">Issues by Type</h3>
                    <canvas id="issuesChart"></canvas>
                </div>
                {{end}}
            </div>

            {{if .ChartData.Heatmap}}
            <div style="margin-top: 2rem;">
                <h3 class="chart-title">📁 Complexity Heatmap</h3>
                <p style="color: var(--text-secondary); font-size: 0.9rem; margin-bottom: 1rem;">
                    Each cell represents a file. Color indicates complexity, size indicates lines of code. Hover for details.
                </p>
                <div class="heatmap-grid">
                    {{range .ChartData.Heatmap}}
                    <div class="heatmap-cell size-{{.Size}}"
                         style="background-color: {{.Color}};"
                         title="{{.FileName}}&#10;Complexity: {{printf "%.1f" .Complexity}}&#10;Lines: {{.LOC}}&#10;Functions: {{.Functions}}">
                        <span class="heatmap-filename">{{.FileName}}</span>
                        <span class="heatmap-complexity">{{printf "%.1f" .Complexity}}</span>
                    </div>
                    {{end}}
                </div>
            </div>
            {{end}}

            {{if .ChartData.DependencyGraph}}
            <div style="margin-top: 2rem;">
                <h3 class="chart-title">🔗 Dependency Graph</h3>
                <p style="color: var(--text-secondary); font-size: 0.9rem; margin-bottom: 1rem;">
                    Interactive package dependency visualization. Circular dependencies highlighted in red.
                </p>
                <div id="dependencyGraph" style="width: 100%; height: 600px; background: var(--bg); border-radius: 0.5rem;"></div>
            </div>
            {{end}}

            {{if .ChartData.MetricsTimeSeries}}
            <div style="margin-top: 2rem;">
                <h3 class="chart-title">📈 Historical Trends</h3>
                <p style="color: var(--text-secondary); font-size: 0.9rem; margin-bottom: 1rem;">
                    Metrics over time showing code quality evolution. Based on {{len .ChartData.MetricsTimeSeries.Labels}} historical reports.
                </p>
                <div class="chart-container">
                    <canvas id="trendsChart"></canvas>
                </div>
            </div>
            {{end}}
        </div>
        {{end}}

        {{if .Metrics.LargestFiles}}
        {{if gt (len .Metrics.LargestFiles) 0}}
        <div class="section">
            <h2 class="section-title">📄 Largest Files</h2>
            <table>
                <thead>
                    <tr>
                        <th>Rank</th>
                        <th>File</th>
                        <th>Lines</th>
                    </tr>
                </thead>
                <tbody>
                    {{range $index, $file := .Metrics.LargestFiles}}
                    <tr>
                        <td>{{add $index 1}}</td>
                        <td><code>{{$file.Path}}</code></td>
                        <td>{{formatNumber $file.Lines}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        {{end}}
        {{end}}

        {{if .Metrics.MostComplexFunctions}}
        {{if gt (len .Metrics.MostComplexFunctions) 0}}
        <div class="section">
            <h2 class="section-title">🎯 Most Complex Functions</h2>
            <table>
                <thead>
                    <tr>
                        <th>Rank</th>
                        <th>Function</th>
                        <th>Complexity</th>
                        <th>Lines</th>
                        <th>File</th>
                    </tr>
                </thead>
                <tbody>
                    {{range $index, $fn := .Metrics.MostComplexFunctions}}
                    <tr>
                        <td>{{add $index 1}}</td>
                        <td><code>{{$fn.Function}}</code></td>
                        <td><strong>{{$fn.Complexity}}</strong></td>
                        <td>{{$fn.Lines}}</td>
                        <td><code>{{$fn.File}}</code></td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        {{end}}
        {{end}}

        <footer>
            <p>Generated by Code Review Assistant</p>
        </footer>
    </div>

    {{if .ChartData}}
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
    {{if .ChartData.DependencyGraph}}
    <script src="https://unpkg.com/vis-network@9.1.9/dist/vis-network.min.js"></script>
    {{end}}
    <script>
        // Chart.js initialization
        const chartData = {{.ChartDataJSON}};

        // Complexity Distribution Chart
        {{if .ChartData.ComplexityDist}}
        const complexityCtx = document.getElementById('complexityChart');
        if (complexityCtx && chartData.complexityDist) {
            new Chart(complexityCtx, {
                type: 'bar',
                data: {
                    labels: chartData.complexityDist.ranges,
                    datasets: [{
                        label: 'Number of Functions',
                        data: chartData.complexityDist.counts,
                        backgroundColor: [
                            '#10b981',  // Green for 1-5
                            '#84cc16',  // Light green for 6-10
                            '#f59e0b',  // Yellow for 11-15
                            '#fb923c',  // Orange for 16-20
                            '#ef4444'   // Red for 20+
                        ],
                        borderColor: [
                            '#059669',
                            '#65a30d',
                            '#d97706',
                            '#ea580c',
                            '#dc2626'
                        ],
                        borderWidth: 1
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: true,
                    plugins: {
                        legend: {
                            display: false
                        },
                        tooltip: {
                            callbacks: {
                                label: function(context) {
                                    return context.parsed.y + ' functions with complexity ' + context.label;
                                }
                            }
                        }
                    },
                    scales: {
                        y: {
                            beginAtZero: true,
                            ticks: {
                                stepSize: 1
                            }
                        }
                    }
                }
            });
        }
        {{end}}

        // Coverage Breakdown Chart
        {{if .ChartData.CoverageBreakdown}}
        const coverageCtx = document.getElementById('coverageChart');
        if (coverageCtx && chartData.coverageBreakdown) {
            new Chart(coverageCtx, {
                type: 'bar',
                data: {
                    labels: chartData.coverageBreakdown.packages,
                    datasets: [{
                        label: 'Coverage %',
                        data: chartData.coverageBreakdown.coverages,
                        backgroundColor: chartData.coverageBreakdown.colors,
                        borderColor: chartData.coverageBreakdown.colors,
                        borderWidth: 1
                    }]
                },
                options: {
                    indexAxis: 'y',
                    responsive: true,
                    maintainAspectRatio: true,
                    plugins: {
                        legend: {
                            display: false
                        },
                        tooltip: {
                            callbacks: {
                                label: function(context) {
                                    return context.parsed.x.toFixed(1) + '% coverage';
                                }
                            }
                        }
                    },
                    scales: {
                        x: {
                            beginAtZero: true,
                            max: 100,
                            ticks: {
                                callback: function(value) {
                                    return value + '%';
                                }
                            }
                        }
                    }
                }
            });
        }
        {{end}}

        // Issues by Type Chart
        {{if .ChartData.IssueCounts}}
        const issuesCtx = document.getElementById('issuesChart');
        if (issuesCtx && chartData.issueCounts) {
            new Chart(issuesCtx, {
                type: 'bar',
                data: {
                    labels: chartData.issueCounts.types,
                    datasets: [
                        {
                            label: 'Errors',
                            data: chartData.issueCounts.errorCount,
                            backgroundColor: '#ef4444',
                            borderColor: '#dc2626',
                            borderWidth: 1
                        },
                        {
                            label: 'Warnings',
                            data: chartData.issueCounts.warnCount,
                            backgroundColor: '#f59e0b',
                            borderColor: '#d97706',
                            borderWidth: 1
                        },
                        {
                            label: 'Info',
                            data: chartData.issueCounts.infoCount,
                            backgroundColor: '#3b82f6',
                            borderColor: '#2563eb',
                            borderWidth: 1
                        }
                    ]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: true,
                    plugins: {
                        legend: {
                            display: true,
                            position: 'top'
                        },
                        tooltip: {
                            mode: 'index',
                            intersect: false
                        }
                    },
                    scales: {
                        x: {
                            stacked: true
                        },
                        y: {
                            stacked: true,
                            beginAtZero: true,
                            ticks: {
                                stepSize: 1
                            }
                        }
                    }
                }
            });
        }
        {{end}}

        // Dependency Graph Visualization
        {{if .ChartData.DependencyGraph}}
        const graphContainer = document.getElementById('dependencyGraph');
        if (graphContainer && chartData.dependencyGraph) {
            // Prepare nodes with styling
            const nodes = new vis.DataSet(chartData.dependencyGraph.nodes.map(node => ({
                id: node.id,
                label: node.label,
                title: node.title,
                value: node.value,
                color: node.group === 'internal' ? '#10b981' : node.group === 'external' ? '#f59e0b' : '#3b82f6',
                font: { color: '#111827' }
            })));

            // Prepare edges with styling
            const edges = new vis.DataSet(chartData.dependencyGraph.edges.map(edge => ({
                from: edge.from,
                to: edge.to,
                arrows: 'to',
                color: edge.color || { color: '#9ca3af', highlight: '#6b7280' },
                width: edge.width || 1,
                smooth: { type: 'cubicBezier' }
            })));

            const data = { nodes, edges };

            const options = {
                nodes: {
                    shape: 'dot',
                    size: 15,
                    font: {
                        size: 12,
                        face: 'Arial',
                        color: '#111827'
                    },
                    borderWidth: 2,
                    borderWidthSelected: 4
                },
                edges: {
                    smooth: {
                        type: 'cubicBezier',
                        forceDirection: 'horizontal',
                        roundness: 0.4
                    }
                },
                physics: {
                    enabled: true,
                    barnesHut: {
                        gravitationalConstant: -3000,
                        centralGravity: 0.3,
                        springLength: 150,
                        springConstant: 0.04,
                        damping: 0.09,
                        avoidOverlap: 0.5
                    },
                    stabilization: {
                        enabled: true,
                        iterations: 200,
                        updateInterval: 25
                    }
                },
                interaction: {
                    hover: true,
                    tooltipDelay: 100,
                    zoomView: true,
                    dragView: true
                },
                layout: {
                    improvedLayout: true,
                    hierarchical: false
                }
            };

            const network = new vis.Network(graphContainer, data, options);

            // Fit the network after stabilization
            network.once('stabilizationIterationsDone', function() {
                network.fit({
                    animation: {
                        duration: 1000,
                        easingFunction: 'easeInOutQuad'
                    }
                });
            });
        }
        {{end}}

        // Historical Trends Chart
        {{if .ChartData.MetricsTimeSeries}}
        const trendsCanvas = document.getElementById('trendsChart');
        if (trendsCanvas && chartData.metricsTimeSeries) {
            const trendsCtx = trendsCanvas.getContext('2d');
            new Chart(trendsCtx, {
                type: 'line',
                data: {
                    labels: chartData.metricsTimeSeries.labels,
                    datasets: [
                        {
                            label: 'Avg Complexity',
                            data: chartData.metricsTimeSeries.complexity,
                            borderColor: '#f59e0b',
                            backgroundColor: 'rgba(245, 158, 11, 0.1)',
                            tension: 0.4,
                            yAxisID: 'y'
                        },
                        {
                            label: 'Coverage %',
                            data: chartData.metricsTimeSeries.coverage,
                            borderColor: '#10b981',
                            backgroundColor: 'rgba(16, 185, 129, 0.1)',
                            tension: 0.4,
                            yAxisID: 'y1'
                        },
                        {
                            label: 'Issues',
                            data: chartData.metricsTimeSeries.issueCount,
                            borderColor: '#ef4444',
                            backgroundColor: 'rgba(239, 68, 68, 0.1)',
                            tension: 0.4,
                            yAxisID: 'y2'
                        },
                        {
                            label: 'Lines of Code',
                            data: chartData.metricsTimeSeries.totalLines,
                            borderColor: '#3b82f6',
                            backgroundColor: 'rgba(59, 130, 246, 0.1)',
                            tension: 0.4,
                            yAxisID: 'y3'
                        }
                    ]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: true,
                    interaction: {
                        mode: 'index',
                        intersect: false
                    },
                    plugins: {
                        legend: {
                            position: 'top',
                            labels: {
                                usePointStyle: true,
                                padding: 15
                            }
                        },
                        tooltip: {
                            callbacks: {
                                title: function(context) {
                                    const index = context[0].dataIndex;
                                    const timestamp = chartData.metricsTimeSeries.timestamps[index];
                                    return new Date(timestamp).toLocaleString();
                                }
                            }
                        }
                    },
                    scales: {
                        x: {
                            grid: {
                                display: false
                            }
                        },
                        y: {
                            type: 'linear',
                            display: true,
                            position: 'left',
                            title: {
                                display: true,
                                text: 'Avg Complexity'
                            },
                            grid: {
                                color: 'rgba(0, 0, 0, 0.05)'
                            }
                        },
                        y1: {
                            type: 'linear',
                            display: true,
                            position: 'right',
                            title: {
                                display: true,
                                text: 'Coverage %'
                            },
                            grid: {
                                drawOnChartArea: false
                            }
                        },
                        y2: {
                            type: 'linear',
                            display: false,
                            position: 'right'
                        },
                        y3: {
                            type: 'linear',
                            display: false,
                            position: 'right'
                        }
                    }
                }
            });
        }
        {{end}}
    </script>
    {{end}}
</body>
</html>`
