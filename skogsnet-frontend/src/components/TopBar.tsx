import { Text } from "@/components/retroui/Text";
import { Checkbox } from "@/components/retroui/Checkbox";
import React from "react";
import type { FC } from "react";

interface TopBarProps {
    darkMode: boolean;
    setDarkMode: React.Dispatch<React.SetStateAction<boolean>>;
    showDataRange: string;
    setShowDataRange: (v: string) => void;
    liveDataChecked: boolean;
    handleLiveDataChecked: (checked: boolean) => void;
    fetchError: string | null;
    TimeRangeSelection: FC<{ value: string; onChange: (v: string) => void }>;
}

const TopBar: FC<TopBarProps> = ({
    darkMode,
    setDarkMode,
    showDataRange,
    setShowDataRange,
    liveDataChecked,
    handleLiveDataChecked,
    fetchError,
    TimeRangeSelection,
}) => (
    <div id="top-bar" className="flex flex-col p-6">
        {/* Desktop */}
        <div className="hidden sm:flex flex-row items-center w-full gap-4">
            <Text as="h4" className="w-auto">Skogsnet</Text>
            <div className="flex gap-2 items-center ml-6">
                <Text as="p" className="">Show: </Text>
                <TimeRangeSelection value={showDataRange} onChange={setShowDataRange} />
            </div>
            <div className="flex gap-2 items-center ml-6">
                <Checkbox
                    checked={liveDataChecked}
                    onCheckedChange={handleLiveDataChecked}
                />
                <Text>Live update</Text>
                {fetchError && (
                    <Text as="p" className="text-sm text-red-600 ml-6">{fetchError}</Text>
                )}
            </div>
            <button
                className="px-3 py-1 rounded border border-gray-400 bg-card text-foreground hover:bg-muted transition ml-auto"
                onClick={() => setDarkMode(d => !d)}
                aria-label="Toggle dark mode"
            >
                {darkMode ? "🌙 Dark" : "☀️ Light"}
            </button>
        </div>
        {/* Mobile */}
        <div className="flex flex-row items-center w-full mb-2 sm:hidden">
            <Text as="h4" className="w-auto">Skogsnet</Text>
            <button
                className="px-3 py-1 rounded border border-gray-400 bg-card text-foreground hover:bg-muted transition ml-auto"
                onClick={() => setDarkMode(d => !d)}
                aria-label="Toggle dark mode"
            >
                {darkMode ? "🌙 Dark" : "☀️ Light"}
            </button>
        </div>
        <div className="flex flex-row items-center w-full gap-2 sm:hidden">
            <div className="flex gap-2 items-center">
                <Text as="p" className="">Show: </Text>
                <TimeRangeSelection value={showDataRange} onChange={setShowDataRange} />
            </div>
            <div className="flex gap-2 items-center ml-auto">
                <Checkbox
                    checked={liveDataChecked}
                    onCheckedChange={handleLiveDataChecked}
                />
                <Text>Live update</Text>
                {fetchError && (
                    <Text as="p" className="text-sm text-red-600 ml-6">{fetchError}</Text>
                )}
            </div>
        </div>
    </div>
);

export default TopBar;