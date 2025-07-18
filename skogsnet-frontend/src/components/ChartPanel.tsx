import React from "react";
import Chart from "react-apexcharts";
import type { Measurement } from "../interfaces/Measurement";

interface ChartPanelProps {
    darkMode: boolean;
    measurements: Measurement[];
    showDataRange: string;
    chartColors: string[];
}

const ChartPanel: React.FC<ChartPanelProps> = ({ darkMode, measurements, chartColors }) => {
    React.useEffect(() => {
        if (darkMode) {
            document.documentElement.classList.add("dark");
        } else {
            document.documentElement.classList.remove("dark");
        }
    }, [darkMode]);

    if (!measurements || measurements.length === 0) {
        return (
            <div className="flex items-center justify-center h-full text-gray-500 text-lg">
                No data to display
            </div>
        );
    }

    const chartOptions: ApexCharts.ApexOptions = {
        chart: {
            type: "line",
            height: "100%",
            width: "100%",
            animations: {
                enabled: true,
            },
            background: darkMode ? "#121212" : "#fff",
        },
        theme: {
            mode: darkMode ? "dark" : "light",
        },
        series: [
            {
                name: "Temperature",
                data: measurements.map(m => m.AvgTemperature),
            },
            {
                name: "Outside temperature",
                data: measurements.map(m => m.AvgWeatherTemp),
            },
            {
                name: "Humidity",
                data: measurements.map(m => m.AvgHumidity),
            },
            {
                name: "Wind Speed",
                data: measurements.map(m => m.AvgWindSpeed),
            }
        ],
        legend: {
            show: true,
            clusterGroupedSeries: false,
            position: "bottom",
            horizontalAlign: "center",
            floating: false,
            fontSize: "14px",
            fontFamily: "Space Grotesk",
            itemMargin: {
                horizontal: 16,
                vertical: 8,
            },
        },
        grid: {
            borderColor: darkMode ? "#27272a" : "#e5e7eb", // darker for dark mode, lighter for light mode
            xaxis: {
                lines: {
                    show: true,
                },
            },
            yaxis: {
                lines: {
                    show: true,
                },
            },
        },
        colors: chartColors,
        stroke: {
            width: [2, 2, 2, 2],
            dashArray: [0, 0, 2, 2],
        },
        xaxis: {
            type: "datetime",
            categories: measurements.map(m => m.AggregatedTimestamp),
            labels: {
                datetimeUTC: false,
                datetimeFormatter: {
                    year: 'yyyy',
                    month: 'MMM \'yy',
                    day: 'dd MMM',
                    hour: 'HH:mm'
                },
            }
        },
        title: {
            text: "Weather Data Over Time",
            align: "center",
            style: {
                fontSize: "16px",
                fontFamily: "Space Grotesk",
                color: darkMode ? "#e5e7eb" : "#121212",

            },
        },
        tooltip: {
            shared: true,
            intersect: false,
            x: {
                format: 'dd MMM HH:mm',
            },
            y: {
                formatter: function (value: number, { seriesIndex }: { seriesIndex: number }) {
                    if (value === undefined) return '--';
                    if (seriesIndex === 0 || seriesIndex === 1) {
                        return `${value.toFixed(1)} °C`;
                    } else if (seriesIndex === 2) {
                        return `${value.toFixed(1)} %`;
                    } else if (seriesIndex === 3) {
                        return `${value.toFixed(1)} m/s`;
                    }
                    return `${value}`;
                },
            },
        },
        yaxis: [
            {
                seriesName: ["Temperature", "Outside temperature"],
                title: {
                    text: "Temperature (°C)",
                    style: { color: chartColors[0] },
                },
                labels: {
                    style: { colors: chartColors[0] },
                    formatter: function (value: number) {
                        return value !== undefined ? `${value.toFixed(1)} °C` : '--';
                    },
                },
            },
            {
                opposite: true,
                title: {
                    text: "Humidity %",
                    style: { color: chartColors[2] },
                },
                labels: {
                    style: { colors: chartColors[2] },
                    formatter: function (value: number) {
                        return value !== undefined ? `${value.toFixed(1)} %` : '--';
                    }
                },
            },
            {
                opposite: true,
                seriesName: "Wind Speed",
                title: {
                    text: "Wind Speed (m/s)",
                    style: { color: chartColors[3] },
                },
                labels: {
                    style: { colors: chartColors[3] },
                    formatter: function (value: number) {
                        return value !== undefined ? `${value.toFixed(1)} m/s` : '--';
                    }
                },
            }
        ],
    };

    return (
        <div className="w-full h-full min-h-[800px] sm:min-h-[400px] mt-4 mb-4">
            <Chart
                options={chartOptions}
                series={chartOptions.series}
                width="100%"
                height="100%"
                type="line"
            />
        </div>
    );
};

export default ChartPanel;