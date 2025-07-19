import { Badge } from "@/components/retroui/Badge";
import type { LatestMeasurementResponse } from "../interfaces/Measurement";

interface DataBarProps {
    data: LatestMeasurementResponse | null;
    darkMode?: boolean;
}

export default function DataBar({
    data,
    darkMode,
}: DataBarProps) {
    if (data == null || data.latest == null) {
        return (
            <div id="data-bar" className="flex flex-wrap items-center gap-4 ml-6 mr-6">
                <Badge size="md" className="w-full sm:w-auto">
                    No data available
                </Badge>
            </div>
        )
    }

    else {
        return (
            <div id="data-bar" className="flex flex-wrap items-center gap-4 ml-6 mr-6">
                <Badge size="md" className="w-full sm:w-auto">
                    Temp: {data.latest.AvgTemperature.toFixed(2)} °C
                </Badge>
                <Badge size="md" className="w-full sm:w-auto">
                    Outside Temp: {data.latest.AvgWeatherTemp !== 0 ? data.latest.AvgWeatherTemp.toFixed(2) : "No data"} °C
                </Badge>
                <Badge size="md" className="w-full sm:w-auto">
                    Humidity: {data.latest.AvgHumidity.toFixed(2)} %
                </Badge>
                <Badge size="md" className="w-full sm:w-auto">
                    Wind Speed: {data.latest.AvgWindSpeed.toFixed(2)} m/s
                </Badge>
                <Badge size="md" className="w-full sm:w-auto">
                    Weather: {data.latest.Description || "No data"}
                </Badge>
                <Badge size="md" className="w-full sm:w-auto">
                    <span
                        className={`${(data.trajectory ?? 0) > 0 ? "text-red-500" : (data.trajectory ?? 0) < 0 ? "text-green-500" : (darkMode ? "text-gray-200" : "text-gray-700")}`}
                    >
                        Δ Temp: {(data.trajectory ?? 0).toFixed(2)} °C
                    </span>
                </Badge>
            </div>
        );

    }
}
