import { Badge } from "@/components/retroui/Badge";
import type { Measurement } from "../interfaces/Measurement";

interface DataBarProps {
    latestMeasurement: Measurement | null;
    currentTemp: number;
    currentHumidity: number;
    currentOutsideTemp: number;
}

export default function DataBar({
    latestMeasurement,
    currentTemp,
    currentHumidity,
    currentOutsideTemp,
}: DataBarProps) {
    return (
        <div id="data-bar" className="flex flex-wrap items-center gap-4 ml-6 mr-6">
            <Badge size="md" className="w-full sm:w-auto">
                Temp: {currentTemp.toFixed(2)} °C
            </Badge>
            <Badge size="md" className="w-full sm:w-auto">
                Outside Temp: {currentOutsideTemp.toFixed(2)} °C
            </Badge>
            <Badge size="md" className="w-full sm:w-auto">
                Humidity: {currentHumidity.toFixed(2)} %
            </Badge>
            <Badge size="md" className="w-full sm:w-auto">
                Wind Speed: {latestMeasurement ? latestMeasurement.AvgWindSpeed.toFixed(2) : 0} m/s
            </Badge>
            <Badge size="md" className="w-full sm:w-auto">
                Weather: {latestMeasurement ? latestMeasurement.Description : 'N/A'}
            </Badge>
        </div>
    );
}