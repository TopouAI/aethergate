type TrafficChartProps = {
  values: number[];
};

export function TrafficChart({ values }: TrafficChartProps) {
  const width = 760;
  const height = 220;
  const padding = 10;
  const max = Math.max(...values) * 1.08;
  const min = Math.min(...values) * 0.82;
  const range = Math.max(max - min, 1);
  const points = values.map((value, index) => {
    const x = padding + (index / Math.max(values.length - 1, 1)) * (width - padding * 2);
    const y = height - padding - ((value - min) / range) * (height - padding * 2);
    return { x, y };
  });
  const line = points.map((point, index) => `${index === 0 ? "M" : "L"}${point.x},${point.y}`).join(" ");
  const area = `${line} L${points.at(-1)?.x ?? width},${height} L${points[0]?.x ?? 0},${height} Z`;

  return (
    <div className="relative h-[230px] w-full overflow-hidden">
      <div className="pointer-events-none absolute inset-0 flex flex-col justify-between py-3">
        {[0, 1, 2, 3, 4].map((lineIndex) => (
          <span key={lineIndex} className="block border-t border-dashed border-white/[0.055]" />
        ))}
      </div>
      <svg viewBox={`0 0 ${width} ${height}`} className="relative h-full w-full overflow-visible" preserveAspectRatio="none" role="img" aria-label="Request volume trend">
        <defs>
          <linearGradient id="trafficArea" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#70a7ff" stopOpacity="0.28" />
            <stop offset="100%" stopColor="#70a7ff" stopOpacity="0" />
          </linearGradient>
          <filter id="trafficGlow" x="-20%" y="-20%" width="140%" height="140%">
            <feGaussianBlur stdDeviation="4" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>
        <path d={area} fill="url(#trafficArea)" />
        <path d={line} fill="none" stroke="#70a7ff" strokeWidth="2.2" vectorEffect="non-scaling-stroke" filter="url(#trafficGlow)" />
        {points.map((point, index) => (
          <circle key={`${point.x}-${point.y}`} cx={point.x} cy={point.y} r={index === points.length - 1 ? 4 : 1.8} fill={index === points.length - 1 ? "#dbeafe" : "#70a7ff"} opacity={index === points.length - 1 ? 1 : 0.52} vectorEffect="non-scaling-stroke" />
        ))}
      </svg>
    </div>
  );
}

