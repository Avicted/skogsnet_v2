import './App.css'
import { useCallback, useEffect, useState, useRef } from 'react';
import ChartPanel from './components/ChartPanel';
import type { LatestMeasurementResponse, Measurement } from './interfaces/Measurement';
import TopBar from "./components/TopBar";
import TimeRangeSelection from "./components/TimeRangeSelection";
import DataBar from "./components/DataBar";

function App() {
  const [darkMode, setDarkMode] = useState<boolean>(() => {
    return localStorage.getItem("darkMode") === "true";
  });

  const [liveDataChecked, setLiveDataChecked] = useState<boolean>(true);
  const [showDataRange, setShowDataRange] = useState<string>('today');
  const [measurements, setMeasurements] = useState<Measurement[]>([]);
  const [latestMeasurement, setLatestMeasurement] = useState<LatestMeasurementResponse | null>(null);
  const [fetchError, setFetchError] = useState<string | null>(null);

  const fetchInterval = 10000;
  const latestFetchController = useRef<AbortController | null>(null);
  const handleLiveDataChecked = useCallback(
    (checked: boolean) => setLiveDataChecked(checked),
    []
  );

  const handleShowDataRange = useCallback(
    (value: string) => setShowDataRange(value),
    []
  );

  const fetchMeasurements = useCallback(
    async ({
      latest = false,
      signal,
    }: { latest?: boolean; signal?: AbortSignal }) => {
      setFetchError(null);
      const url = latest
        ? "http://localhost:8080/api/measurements/latest"
        : `http://localhost:8080/api/measurements?range=${showDataRange}`;

      try {
        const response = await fetch(url, { signal });
        if (!response.ok) throw new Error('Network response was not ok');
        const data = await response.json();

        if (latest) {
          setLatestMeasurement(data);
          return data;
        } else if (Array.isArray(data)) {
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
    },
    [showDataRange]
  );

  const chartColors = ["#ef4444", "#ffae00ff", "#3b82f6", "#ff00ff"]

  useEffect(() => {
    if (latestFetchController.current) {
      latestFetchController.current.abort();
    }
    const controller = new AbortController();
    latestFetchController.current = controller;

    const fetchAll = async () => {
      await fetchMeasurements({ latest: true, signal: controller.signal });
      if (liveDataChecked) {
        await fetchMeasurements({ latest: false, signal: controller.signal });
      }
    };

    fetchAll();

    const interval = setInterval(fetchAll, fetchInterval);

    return () => {
      clearInterval(interval);
      controller.abort();
    };
  }, [liveDataChecked, showDataRange, fetchMeasurements]);

  useEffect(() => {
    if (darkMode) {
      document.documentElement.classList.add("dark");
    } else {
      document.documentElement.classList.remove("dark");
    }
    localStorage.setItem("darkMode", darkMode ? "true" : "false");
  }, [darkMode]);

  return (
    <div className="flex flex-col h-screen">
      <TopBar
        darkMode={darkMode}
        setDarkMode={setDarkMode}
        showDataRange={showDataRange}
        setShowDataRange={handleShowDataRange}
        liveDataChecked={liveDataChecked}
        handleLiveDataChecked={handleLiveDataChecked}
        fetchError={fetchError}
        TimeRangeSelection={TimeRangeSelection}
      />

      <DataBar
        darkMode={darkMode}
        data={latestMeasurement}
      />

      <ChartPanel
        darkMode={darkMode}
        measurements={measurements}
        showDataRange={showDataRange}
        chartColors={chartColors}
      />
    </div>
  )
}

export default App
