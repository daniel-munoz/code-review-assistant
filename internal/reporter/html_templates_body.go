package reporter

// htmlTemplateBody contains the main HTML body structure with all dashboard sections
const htmlTemplateBody = `<body>
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
`
