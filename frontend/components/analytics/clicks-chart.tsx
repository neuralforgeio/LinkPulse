"use client";

import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { TimePoint } from "./types";

// Dates arrive as "YYYY-MM-DD" (UTC buckets from the backend).
function shortDate(date: string): string {
  return new Date(date + "T00:00:00Z").toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    timeZone: "UTC",
  });
}

function fullDate(date: string): string {
  return new Date(date + "T00:00:00Z").toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    timeZone: "UTC",
  });
}

// Tick label color that stays readable on both light and dark charts.
const TICK = { fontSize: 11, fill: "#a1a1aa" };

interface ClicksChartProps {
  data: TimePoint[];
  height?: number;
}

/** Area chart of clicks over time. */
export function ClicksChart({ data, height = 280 }: ClicksChartProps) {
  if (data.length === 0) {
    return (
      <div className="flex items-center justify-center rounded-lg border border-dashed border-zinc-300 p-10 text-sm text-zinc-400 dark:border-zinc-700">
        No clicks in this period yet.
      </div>
    );
  }

  return (
    <div style={{ width: "100%", height }}>
      <ResponsiveContainer>
        <AreaChart
          data={data}
          margin={{ top: 8, right: 8, left: 0, bottom: 0 }}
        >
          <defs>
            <linearGradient id="lpClicksFill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#3b82f6" stopOpacity={0.28} />
              <stop offset="100%" stopColor="#3b82f6" stopOpacity={0.02} />
            </linearGradient>
          </defs>
          <CartesianGrid
            strokeDasharray="3 3"
            vertical={false}
            stroke="currentColor"
            className="text-zinc-200 dark:text-zinc-800"
          />
          <XAxis
            dataKey="date"
            tickFormatter={shortDate}
            tick={TICK}
            tickLine={false}
            axisLine={false}
            minTickGap={16}
          />
          <YAxis
            allowDecimals={false}
            tick={TICK}
            tickLine={false}
            axisLine={false}
            width={36}
          />
          <Tooltip
            cursor={{ stroke: "#a1a1aa", strokeDasharray: "3 3" }}
            formatter={(value) => [`${value} clicks`, "Clicks"]}
            labelFormatter={(label) => fullDate(String(label))}
            contentStyle={{
              borderRadius: 10,
              border: "1px solid #e4e4e7",
              fontSize: 12,
              boxShadow: "0 4px 12px rgb(0 0 0 / 0.06)",
            }}
            labelStyle={{ color: "#18181b", fontWeight: 600, marginBottom: 2 }}
            itemStyle={{ color: "#71717a" }}
          />
          <Area
            type="monotone"
            dataKey="clicks"
            stroke="#3b82f6"
            strokeWidth={2}
            fill="url(#lpClicksFill)"
            activeDot={{ r: 4, strokeWidth: 0 }}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
