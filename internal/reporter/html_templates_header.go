package reporter

// htmlTemplateHeader contains the HTML head section including CSS styles
const htmlTemplateHeader = `<!DOCTYPE html>
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
`
