document.addEventListener('DOMContentLoaded', function () {
    let chartInstance = null;
    let liveInterval = null;

    async function fetchData(range) {
        const res = await fetch('/api/measurements?range=' + (range || ''));
        return await res.json();
    }

    function format24h(ts) {
        const d = new Date(ts);
        return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false });
    }

    async function drawChart() {
        const range = document.getElementById('range').value;
        const data = await fetchData(range);
        const labels = data.map(m => format24h(m.UnixTimestamp));
        const temp = data.map(m => m.TemperatureCelsius);
        const hum = data.map(m => m.HumidityPercentage);

        const ctx = document.getElementById('chart').getContext('2d');
        const chartOptions = {
            responsive: true,
            maintainAspectRatio: false,
            animation: false,
            plugins: {
                legend: {
                    labels: { color: "#e0e0e0", font: { size: 16 } }
                },
                title: { display: false }
            },
            scales: {
                x: { ticks: { color: "#e0e0e0" }, grid: { color: "#333" } },
                y: {
                    type: 'linear',
                    position: 'left',
                    title: { display: true, text: 'Temperature (°C)', color: '#ff7043' },
                    ticks: {
                        color: "#ff7043",
                        stepSize: 0.1 // increments of 0.1°C
                    },
                    grid: { color: "#333" }
                },
                y1: {
                    type: 'linear',
                    position: 'right',
                    title: { display: true, text: 'Humidity (%)', color: '#42a5f5' },
                    ticks: { color: "#42a5f5" },
                    grid: { drawOnChartArea: false }
                }
            }
        };
        if (chartInstance) {
            chartInstance.data.labels = labels;
            chartInstance.data.datasets[0].data = temp;
            chartInstance.data.datasets[1].data = hum;
            chartInstance.update();
        } else {
            chartInstance = new Chart(ctx, {
                type: 'line',
                data: {
                    labels,
                    datasets: [
                        {
                            label: 'Temperature (°C)',
                            data: temp,
                            borderColor: '#ff7043',
                            backgroundColor: '#ff7043',
                            fill: false,
                            tension: 0.4,
                            yAxisID: 'y'
                        },
                        {
                            label: 'Humidity (%)',
                            data: hum,
                            borderColor: '#42a5f5',
                            backgroundColor: '#42a5f5',
                            fill: false,
                            tension: 0.2,
                            yAxisID: 'y1'
                        }
                    ]
                },
                options: chartOptions
            });
        }
    }

    function startLiveUpdates() {
        if (liveInterval) clearInterval(liveInterval);
        liveInterval = setInterval(drawChart, 10000);
    }

    function stopLiveUpdates() {
        if (liveInterval) clearInterval(liveInterval);
    }

    document.getElementById('range').addEventListener('change', drawChart);
    document.getElementById('liveToggle').addEventListener('change', (e) => {
        if (e.target.checked) {
            startLiveUpdates();
        } else {
            stopLiveUpdates();
        }
    });

    window.addEventListener('resize', drawChart);

    drawChart();
    if (document.getElementById('liveToggle').checked) {
        startLiveUpdates();
    }
});
