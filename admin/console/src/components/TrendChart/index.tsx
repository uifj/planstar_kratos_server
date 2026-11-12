import { Tooltip } from 'antd';
import React from 'react';

export type TrendPoint = { date?: string; count?: number };

/**
 * 零依赖的 14 天柱状趋势图。pro-card 只带 StatisticCard，没有迷你图组件，
 * 引 @ant-design/plots 只为四张小图不值得；SVG/div 柱 + antd Tooltip 足够。
 */
const TrendChart: React.FC<{
  points: TrendPoint[];
  color?: string;
  height?: number;
}> = ({ points, color = '#1C66E5', height = 72 }) => {
  const normalized = points.map((p) => ({
    date: p.date ?? '',
    count: p.count ?? 0,
  }));
  const max = Math.max(1, ...normalized.map((p) => p.count));
  return (
    <div style={{ display: 'flex', alignItems: 'flex-end', gap: 3, height }}>
      {normalized.map((point) => (
        <Tooltip key={point.date} title={`${point.date} · ${point.count}`}>
          <div
            style={{
              flex: 1,
              height: '100%',
              display: 'flex',
              alignItems: 'flex-end',
            }}
          >
            <div
              style={{
                width: '100%',
                height: `${(point.count / max) * 100}%`,
                minHeight: point.count > 0 ? 3 : 1,
                background: point.count > 0 ? color : 'rgba(128,128,128,0.25)',
                borderRadius: 2,
              }}
            />
          </div>
        </Tooltip>
      ))}
    </div>
  );
};

export default TrendChart;