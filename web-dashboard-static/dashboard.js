document.addEventListener('DOMContentLoaded', function () {
    let chartInstance = null;
    let liveInterval = null;

    async function fetchLatest() {
        try {
            const res = await fetch('/api/measurements/latest');
            if (!res.ok) throw new Error(`API error: ${res.status} ${res.statusText}`);
            const data = await res.json();
            if (data && data.length > 0) {
                const latest = data[0];
                document.getElementById('current-temp').textContent = `Temp: ${latest.AvgTemperature.toFixed(2)} °C`;
                document.getElementById('outside-temp').textContent = `Outside Temp: ${latest.AvgWeatherTemp !== 0 ? latest.AvgWeatherTemp.toFixed(2) : '--'} °C`;
                document.getElementById('current-hum').textContent = `Humidity: ${latest.AvgHumidity.toFixed(2)} %`;
            } else {
                document.getElementById('current-temp').textContent = "Temp: -- °C";
                document.getElementById('outside-temp').textContent = "Outside Temp: -- °C";
                document.getElementById('current-hum').textContent = "Humidity: -- %";
            }
        } catch (err) {
            document.getElementById('current-temp').textContent = "Temp: -- °C";
            document.getElementById('outside-temp').textContent = "Outside Temp: -- °C";
            document.getElementById('current-hum').textContent = "Humidity: -- %";
        }
    }

    async function fetchData(range) {
        const errorMessage = document.getElementById('error-message');
        const currentValues = document.getElementById('current-values');

        try {
            const res = await fetch('/api/measurements?range=' + (range || ''));
            if (!res.ok) {
                throw new Error(`API error: ${res.status} ${res.statusText}`);
            }

            errorMessage.style.display = 'none';
            errorMessage.textContent = '';
            currentValues.style.display = 'block';

            return await res.json();
        } catch (err) {

            if (errorMessage) {
                errorMessage.textContent = "Error loading data: " + err.message + ". Retrying...";
                errorMessage.style.display = 'block';
                currentValues.style.display = 'none';
            }

            return null; // Return null to indicate failure
        }
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

        // If fetch failed, try again after a short delay
        if (data === null) {
            setTimeout(drawChart, 5000);
            return;
        }

        const labels = data.map(m => m.AggregatedTimestamp);
        const temp = movingAverage(data.map(m => m.AvgTemperature), 5);
        const hum = movingAverage(data.map(m => m.AvgHumidity), 5);
        const weatherTemp = data.map(m => m.AvgWeatherTemp !== 0 ? m.AvgWeatherTemp : null);

        // Fix for blurry tooltips/lines on high-DPI screens
        const canvas = document.getElementById('chart');
        const dpr = window.devicePixelRatio || 1;
        canvas.width = canvas.offsetWidth * dpr;
        canvas.height = canvas.offsetHeight * dpr;
        canvas.getContext('2d').setTransform(1, 0, 0, 1, 0, 0);
        canvas.getContext('2d').scale(dpr, dpr);

        const ctx = document.getElementById('chart').getContext('2d');

        const chartOptions = {
            responsive: true,
            maintainAspectRatio: false,
            animation: false,
            plugins: {
                legend: {
                    labels: { color: "#e0e0e0", font: { size: 16 } }
                },
                title: { display: false },
                tooltip: {
                    mode: 'index',
                    intersect: false,
                    backgroundColor: '#23272f',
                    borderColor: '#ffb74d',
                    borderWidth: 2,
                    titleColor: '#ffb74d',
                    bodyColor: '#e0e0e0',
                    cornerRadius: 8,
                    padding: 12,
                    titleFont: { size: 16, weight: 'bold' },
                    bodyFont: { size: 15 },
                    displayColors: false, // Hide small color boxes
                    caretSize: 8,
                    boxPadding: 6,
                }
            },
            scales: {
                x: {
                    type: 'time',
                    time: {
                        unit: 'minute',
                        tooltipFormat: 'HH:mm',
                        displayFormats: {
                            minute: 'HH:mm',
                            hour: 'HH:mm'
                        },
                        tooltipFormat: 'HH:mm' // Tooltip format
                    },
                    ticks: {
                        stepSize: 1,
                        autoSkip: false,
                        color: "#e0e0e0",
                        maxRotation: 0,
                        callback: function (value) {
                            // Depending on the range which can be 1h, 6h, 12h, 24h, today, this week, this month, this year, all
                            // show every 5 minutes for 1h
                            // show every 15 minutes for 6h
                            // show every 30 minutes for 12h
                            // show every hour for 24h, today
                            // show every 2 hours for this week
                            // show every 6 hours for this month
                            // show this year by date, not time
                            // show all by automatically adjusting based on the data, by date, not time

                            const date = new Date(value);
                            const minutes = date.getMinutes();
                            const hours = date.getHours();
                            const day = date.getDate();
                            const month = date.getMonth() + 1; // Months are 0-indexed
                            const year = date.getFullYear();
                            const rangeValue = document.getElementById('range').value;

                            if (rangeValue === '1h') {
                                return (minutes % 5 === 0 || minutes === 0) ? format24h(value) : '';
                            } else if (rangeValue === '6h') {
                                return (minutes % 5 === 0 || minutes === 0) ? format24h(value) : '';
                            } else if (rangeValue === '12h') {
                                return (minutes % 30 === 0 || minutes === 0) ? format24h(value) : '';
                            } else if (rangeValue === '24h' || rangeValue === 'today') {
                                return (minutes === 0) ? format24h(value) : '';
                            } else if (rangeValue === 'this_week') {
                                return (hours % 2 === 0 && minutes === 0) ? format24h(value) : '';
                            } else if (rangeValue === 'this_month') {
                                return (hours % 6 === 0 && minutes === 0) ? format24h(value) : '';
                            } else if (rangeValue === 'this_year') {
                                return `${day}/${month}`;
                            }
                            else if (rangeValue === 'all') {
                                return `${day}/${month}`;
                            } else {
                                // For 'all' or any other range, show date only
                                return `${day}/${month}`;
                            }
                        },

                    },
                    grid: { color: "#333" }
                },
                y: {
                    type: 'linear',
                    position: 'left',
                    title: { display: true, text: 'Temp (°C)', color: '#ff7043' },
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
                },
            }
        };
        if (chartInstance) {
            chartInstance.data.labels = labels;
            chartInstance.data.datasets[0].data = temp;
            chartInstance.data.datasets[1].data = hum;
            chartInstance.data.datasets[2].data = weatherTemp;
            chartInstance.update();
        } else {
            const crosshairPlugin = {
                id: 'crosshair',
                afterDatasetsDraw(chart) {
                    const tooltip = chart.tooltip;
                    if (
                        tooltip &&
                        tooltip.opacity !== 0 &&
                        tooltip.dataPoints &&
                        tooltip.dataPoints.length
                    ) {
                        const ctx = chart.ctx;
                        const x = tooltip.dataPoints[0].element.x;

                        ctx.save();
                        ctx.beginPath();
                        ctx.moveTo(x, chart.chartArea.top);
                        ctx.lineTo(x, chart.chartArea.bottom);
                        ctx.lineWidth = 2;
                        ctx.strokeStyle = '#fafafa';
                        ctx.setLineDash([4, 4]);
                        ctx.stroke();
                        ctx.restore();
                    }
                }
            };



            chartInstance = new Chart(ctx, {
                type: 'line',
                data: {
                    labels,
                    datasets: [
                        {
                            label: 'Temp (°C)',
                            data: labels.map((timestamp, i) => ({ x: timestamp, y: temp[i] })),
                            borderColor: '#ff7043',
                            backgroundColor: '#ff7043',
                            fill: false,
                            tension: 0.4,
                            yAxisID: 'y',
                            pointRadius: 0,
                            borderWidth: 1
                        },
                        {
                            label: 'Humidity (%)',
                            data: labels.map((timestamp, i) => ({ x: timestamp, y: hum[i] })),
                            borderColor: '#42a5f5',
                            backgroundColor: '#42a5f5',
                            fill: false,
                            tension: 0.2,
                            yAxisID: 'y1',
                            pointRadius: 0,
                            borderWidth: 1
                        },
                        {
                            label: 'Outside Temp (°C)',
                            data: labels.map((timestamp, i) => ({ x: timestamp, y: weatherTemp[i] })),
                            borderColor: '#ffd600',
                            backgroundColor: '#ffd600',
                            fill: false,
                            tension: 0.1,
                            yAxisID: 'y',
                            pointRadius: 0,
                            borderWidth: 1
                        }
                    ]
                },
                options: chartOptions,
                plugins: [crosshairPlugin]
            });


            // Remove crosshair when mouse leaves
            canvas.addEventListener('mouseleave', function () {
                chartInstance._active = [];
                chartInstance.update();
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

    // Initial fetch of the latest data
    fetchLatest();
    // Periodic update of the latest data
    setInterval(fetchLatest, 10000); // update every 10 seconds
});
