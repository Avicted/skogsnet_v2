import { Select } from "../components/retroui/Select";

interface TimeRangeSelectionProps {
    value: string;
    onChange: (v: string) => void;
}

export default function TimeRangeSelection({ value, onChange }: TimeRangeSelectionProps) {
    return (
        <Select value={value} onValueChange={onChange}>
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
                    <Select.Item value="today">Today</Select.Item>
                    <Select.Item value="week">Week</Select.Item>
                    <Select.Item value="month">Month</Select.Item>
                    <Select.Item value="year">Year</Select.Item>
                </Select.Group>
            </Select.Content>
        </Select>
    );
}