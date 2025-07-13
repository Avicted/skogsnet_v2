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

    function movingAverage(arr, windowSize) {
        if (windowSize <= 1) return arr;
        return arr.map((_, i, a) => {
            const start = Math.max(0, i - windowSize + 1);
            const window = a.slice(start, i + 1);
            return window.reduce((sum, v) => sum + v, 0) / window.length;
        });
    }

    async function drawChart() {
        const range = document.getElementById('range').value;
        const data = await fetchData(range);

        // Show current temp/humidity (latest data point)
        if (data.length > 0) {
            const latest = data[data.length - 1];
            document.getElementById('current-temp').textContent = `Temperature: ${latest.TemperatureCelsius.toFixed(2)} °C`;
            document.getElementById('current-hum').textContent = `Humidity: ${latest.HumidityPercentage.toFixed(2)} %`;
        } else {
            document.getElementById('current-temp').textContent = "Temperature: -- °C";
            document.getElementById('current-hum').textContent = "Humidity: -- %";
        }

        const labels = data.map(m => format24h(m.UnixTimestamp));
        const temp = movingAverage(data.map(m => m.TemperatureCelsius), 5);
        const hum = movingAverage(data.map(m => m.HumidityPercentage), 5);

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
                            yAxisID: 'y',
                            pointRadius: 0
                        },
                        {
                            label: 'Humidity (%)',
                            data: hum,
                            borderColor: '#42a5f5',
                            backgroundColor: '#42a5f5',
                            fill: false,
                            tension: 0.2,
                            yAxisID: 'y1',
                            pointRadius: 0
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
