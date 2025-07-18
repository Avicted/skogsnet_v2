import './App.css'
import { Text } from "@/components/retroui/Text";
import { Select } from './components/retroui/Select';
import { Checkbox } from './components/retroui/Checkbox';
import { Badge } from "@/components/retroui/Badge";
import { useEffect, useState, useRef } from 'react';
import ChartPanel from './components/ChartPanel';
import type { Measurement } from './interfaces/Measurement';

function App() {
  const [liveDataChecked, setSetLiveDataChecked] = useState<boolean>(true);
  const [showDataRange, setShowDataRange] = useState<string>('today');
  const [measurements, setMeasurements] = useState<Measurement[]>([]);
  const [latestMeasurement, setLatestMeasurement] = useState<Measurement | null>(null);
  const [fetchError, setFetchError] = useState<string | null>(null);

  const fetchInterval = 10000;
  const latestFetchController = useRef<AbortController | null>(null);

  const fetchLatestData = async (signal?: AbortSignal) => {
    setFetchError(null);

    try {
      const response = await fetch('http://localhost:8080/api/measurements/latest', { signal });
      if (!response.ok) {
        throw new Error('Network response was not ok');
      }

      const data = await response.json();
      if (Array.isArray(data) && data.length > 0) {
        setLatestMeasurement(data[0]);
        return data;
      } else {
        throw new Error('Invalid data format');
      }
    } catch (error) {
      if ((error as any).name !== "AbortError") {
        setFetchError(error instanceof Error ? error.message : String(error));
      }
    }
  }

  const fetchData = async (signal?: AbortSignal) => {
    setFetchError(null);

    try {
      const response = await fetch(`http://localhost:8080/api/measurements?range=${showDataRange}`, { signal });
      if (!response.ok) {
        throw new Error('Network response was not ok');
      }

      const data = await response.json();
      if (Array.isArray(data)) {
        setMeasurements(data);
        return data;
      } else {
        throw new Error('Invalid data format');
      }
    } catch (error) {
      if ((error as any).name !== "AbortError") {
        setFetchError(error instanceof Error ? error.message : String(error));
      }
    }
  }

  const currentTemp = latestMeasurement ? latestMeasurement.AvgTemperature : 0;
  const currentHumidity = latestMeasurement ? latestMeasurement.AvgHumidity : 0;
  const currentOutsideTemp = latestMeasurement ? (latestMeasurement.AvgWeatherTemp !== 0 ? latestMeasurement.AvgWeatherTemp : 0) : 0;

  const chartColors = ["#ef4444", "#ffae00ff", "#3b82f6", "#ff00ff"]

  useEffect(() => {
    // Abort any ongoing fetches
    if (latestFetchController.current) {
      latestFetchController.current.abort();
    }
    const controller = new AbortController();
    latestFetchController.current = controller;

    const fetchAndSetLatestData = async () => {
      await fetchLatestData(controller.signal);
    };

    const fetchAndSetData = async () => {
      if (liveDataChecked) {
        await fetchData(controller.signal);
      }
    };

    fetchAndSetLatestData();
    fetchAndSetData();

    const interval = setInterval(() => {
      if (liveDataChecked) {
        fetchAndSetLatestData();
        fetchAndSetData();
      }
    }, fetchInterval);

    return () => {
      clearInterval(interval);
      controller.abort();
    };
  }, [liveDataChecked, showDataRange]);

  return (
    <div className="flex flex-col h-screen">
      <div id="top-bar" className="flex flex-wrap items-center p-6 gap-4">
        <Text as="h4" className="mr-12 w-full sm:w-auto">Skogsnet Dashboard</Text>
        <div className="mr-12 flex gap-2 items-center w-full sm:w-auto">
          <Text as="p" className="">Show: </Text>
          <Select value={showDataRange} onValueChange={(value) => setShowDataRange(value)}>
            <Select.Trigger>
              <Select.Value placeholder="Select data range" />
            </Select.Trigger>
            <Select.Content>
              <Select.Group>
                <Select.Item value="all">All</Select.Item>
                <Select.Item value="1h">1h</Select.Item>
                <Select.Item value="6h">6h</Select.Item>
                <Select.Item value="12h">12h</Select.Item>
                <Select.Item value="24h">24h</Select.Item>
                <Select.Item value="today">today</Select.Item>
                <Select.Item value="week">week</Select.Item>
                <Select.Item value="month">month</Select.Item>
                <Select.Item value="year">year</Select.Item>
              </Select.Group>
            </Select.Content>
          </Select>
        </div>
        <div className="flex gap-2 items-center w-full sm:w-auto">
          <Checkbox
            checked={liveDataChecked}
            onCheckedChange={(checked: boolean) => setSetLiveDataChecked(checked)}
          />
          <Text>Live updates</Text>
          {fetchError && (
            <Text as="p" className="text-sm text-red-600 ml-6">{fetchError}</Text>
          )}
        </div>
      </div>
      <div id="data-bar" className="flex flex-wrap items-center gap-4 ml-6 mr-6">
        <Badge size="md" className="w-full sm:w-auto">
          Temp: {currentTemp.toFixed(2)} °C
        </Badge>
        <Badge size="md" className="w-full sm:w-auto">
          Humidity: {currentHumidity.toFixed(2)} %
        </Badge>
        <Badge size="md" className="w-full sm:w-auto">
          Outside Temp: {currentOutsideTemp.toFixed(2)} °C
        </Badge>
        <Badge size="md" className="w-full sm:w-auto">
          Wind Speed: {latestMeasurement ? latestMeasurement.AvgWindSpeed.toFixed(2) : 0} m/s
        </Badge>
        <Badge size="md" className="w-full sm:w-auto">
          Weather: {latestMeasurement ? latestMeasurement.Description : 'N/A'}
        </Badge>
      </div>
      <div id="chart" className="flex-1 w-full h-full mt-4 mb-4">
        <ChartPanel
          measurements={measurements}
          showDataRange={showDataRange}
          chartColors={chartColors}
        />
      </div>
    </div>
  )
}

export default App
