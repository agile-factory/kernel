import type { Lease } from '../../types';
import { formatShortDate } from '../../utils/formatDate';

interface LeaseTimelineProps {
  lease: Lease;
}

export default function LeaseTimeline({ lease }: LeaseTimelineProps) {
  const start = new Date(lease.start_date).getTime();
  const end = new Date(lease.end_date).getTime();
  const now = Date.now();
  const totalDuration = end - start;
  const elapsed = Math.max(0, Math.min(now - start, totalDuration));
  const progress = totalDuration > 0 ? (elapsed / totalDuration) * 100 : 0;

  const isExpired = now > end;
  const isNotStarted = now < start;

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-sm text-gray-500">
        <span>{formatShortDate(lease.start_date)}</span>
        <span>{formatShortDate(lease.end_date)}</span>
      </div>
      <div className="relative h-3 bg-gray-200 rounded-full overflow-hidden">
        <div
          className={`absolute inset-y-0 left-0 rounded-full transition-all ${
            isExpired ? 'bg-red-500' : isNotStarted ? 'bg-gray-300' : 'bg-emerald-500'
          }`}
          style={{ width: `${Math.min(progress, 100)}%` }}
        />
        {!isExpired && !isNotStarted && (
          <div
            className="absolute top-1/2 -translate-y-1/2 w-3 h-3 bg-white border-2 border-emerald-600 rounded-full shadow"
            style={{ left: `calc(${Math.min(progress, 100)}% - 6px)` }}
          />
        )}
      </div>
      <div className="text-center text-xs text-gray-500">
        {isExpired
          ? 'Bail expiré'
          : isNotStarted
          ? 'Bail non démarré'
          : `${Math.round(progress)}% écoulé`}
      </div>
    </div>
  );
}
