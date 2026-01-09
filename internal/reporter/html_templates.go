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

        .empty-state {
            text-align: center;
            padding: 3rem;
            color: var(--text-secondary);
        }

        .empty-state-icon {
            font-size: 3rem;
            margin-bottom: 1rem;
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
        {{else}}
        <div class="section">
            <div class="empty-state">
                <div class="empty-state-icon">✅</div>
                <h3>No Issues Found</h3>
                <p>Great job! No code quality issues were detected.</p>
            </div>
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

        <footer>
            <p>Generated by Code Review Assistant</p>
        </footer>
    </div>
</body>
</html>`
