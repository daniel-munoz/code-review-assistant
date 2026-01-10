package reporter

// htmlTemplateScripts contains the JavaScript code for charts and visualizations
const htmlTemplateScripts = `
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
</html>
`
