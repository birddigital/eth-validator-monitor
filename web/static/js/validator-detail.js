/**
 * Validator Detail Page - Chart.js Initialization
 * Handles rendering of effectiveness and attestation charts
 */

document.addEventListener('DOMContentLoaded', function() {
    const chartDataElement = document.getElementById('chart-data');

    if (!chartDataElement) {
        console.warn('Chart data element not found');
        return;
    }

    // Parse chart data from data attributes
    let effectivenessData = [];
    let attestationData = [];

    try {
        const effectivenessRaw = chartDataElement.getAttribute('data-effectiveness');
        const attestationsRaw = chartDataElement.getAttribute('data-attestations');

        effectivenessData = effectivenessRaw ? JSON.parse(effectivenessRaw) : [];
        attestationData = attestationsRaw ? JSON.parse(attestationsRaw) : [];
    } catch (error) {
        console.error('Failed to parse chart data:', error);
        return;
    }

    // Initialize Effectiveness Chart (Line Chart)
    const effectivenessCanvas = document.getElementById('effectiveness-chart');
    if (effectivenessCanvas && effectivenessData.length > 0) {
        const effectivenessCtx = effectivenessCanvas.getContext('2d');

        new Chart(effectivenessCtx, {
            type: 'line',
            data: {
                labels: effectivenessData.map(point => {
                    const date = new Date(point.date);
                    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
                }),
                datasets: [{
                    label: 'Effectiveness (%)',
                    data: effectivenessData.map(point => (point.effectiveness * 100).toFixed(2)),
                    borderColor: 'rgb(59, 130, 246)',
                    backgroundColor: 'rgba(59, 130, 246, 0.1)',
                    borderWidth: 2,
                    fill: true,
                    tension: 0.4,
                    pointRadius: 4,
                    pointHoverRadius: 6,
                    pointBackgroundColor: 'rgb(59, 130, 246)',
                    pointBorderColor: '#fff',
                    pointBorderWidth: 2
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: {
                    mode: 'index',
                    intersect: false
                },
                plugins: {
                    legend: {
                        display: true,
                        position: 'top'
                    },
                    tooltip: {
                        backgroundColor: 'rgba(0, 0, 0, 0.8)',
                        padding: 12,
                        titleFont: {
                            size: 14,
                            weight: 'bold'
                        },
                        bodyFont: {
                            size: 13
                        },
                        callbacks: {
                            label: function(context) {
                                return 'Effectiveness: ' + context.parsed.y + '%';
                            }
                        }
                    }
                },
                scales: {
                    y: {
                        beginAtZero: false,
                        min: 80,
                        max: 100,
                        ticks: {
                            callback: function(value) {
                                return value + '%';
                            }
                        },
                        grid: {
                            color: 'rgba(156, 163, 175, 0.1)'
                        }
                    },
                    x: {
                        grid: {
                            display: false
                        }
                    }
                }
            }
        });
    } else if (effectivenessCanvas) {
        effectivenessCanvas.parentElement.innerHTML = '<p class="text-center text-gray-500 dark:text-gray-400">No effectiveness data available</p>';
    }

    // Initialize Attestation Stats Chart (Bar Chart)
    const attestationCanvas = document.getElementById('attestation-chart');
    if (attestationCanvas && attestationData.length > 0) {
        const attestationCtx = attestationCanvas.getContext('2d');

        new Chart(attestationCtx, {
            type: 'bar',
            data: {
                labels: attestationData.map(stat => {
                    const date = new Date(stat.month);
                    return date.toLocaleDateString('en-US', { year: 'numeric', month: 'short' });
                }),
                datasets: [
                    {
                        label: 'Successful',
                        data: attestationData.map(stat => stat.successful_count),
                        backgroundColor: 'rgba(34, 197, 94, 0.8)',
                        borderColor: 'rgb(34, 197, 94)',
                        borderWidth: 1
                    },
                    {
                        label: 'Missed',
                        data: attestationData.map(stat => stat.missed_count),
                        backgroundColor: 'rgba(239, 68, 68, 0.8)',
                        borderColor: 'rgb(239, 68, 68)',
                        borderWidth: 1
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: {
                    mode: 'index',
                    intersect: false
                },
                plugins: {
                    legend: {
                        display: true,
                        position: 'top'
                    },
                    tooltip: {
                        backgroundColor: 'rgba(0, 0, 0, 0.8)',
                        padding: 12,
                        titleFont: {
                            size: 14,
                            weight: 'bold'
                        },
                        bodyFont: {
                            size: 13
                        },
                        callbacks: {
                            label: function(context) {
                                const label = context.dataset.label || '';
                                const value = context.parsed.y;
                                const total = attestationData[context.dataIndex].total_count;
                                const percentage = ((value / total) * 100).toFixed(2);
                                return label + ': ' + value + ' (' + percentage + '%)';
                            },
                            footer: function(tooltipItems) {
                                const dataIndex = tooltipItems[0].dataIndex;
                                const total = attestationData[dataIndex].total_count;
                                return 'Total: ' + total;
                            }
                        }
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        stacked: false,
                        ticks: {
                            precision: 0
                        },
                        grid: {
                            color: 'rgba(156, 163, 175, 0.1)'
                        }
                    },
                    x: {
                        stacked: false,
                        grid: {
                            display: false
                        }
                    }
                }
            }
        });
    } else if (attestationCanvas) {
        attestationCanvas.parentElement.innerHTML = '<p class="text-center text-gray-500 dark:text-gray-400">No attestation data available</p>';
    }

    // Handle SSE updates for charts (refresh charts when data updates)
    setupChartRefresh();
});

/**
 * Setup SSE listener to refresh charts when validator data updates
 */
function setupChartRefresh() {
    // Listen for HTMX SSE events
    document.body.addEventListener('htmx:sseMessage', function(event) {
        if (event.detail.type === 'validator-update') {
            // When validator metadata updates, we might want to refresh charts
            // For now, we'll just log it. In production, you might want to
            // fetch new chart data and re-render the charts
            console.log('Validator data updated via SSE');
        }
    });
}
