import { PieChart, Pie, Cell, ResponsiveContainer } from 'recharts';
import type { DashboardStats } from '../../types';
import Card from '../ui/Card';

interface OccupancyRateProps {
  stats?: DashboardStats;
}

export default function OccupancyRate({ stats }: OccupancyRateProps) {
  const occupied = stats?.occupied_properties || 0;
  const total = stats?.total_properties || 1;
  const available = total - occupied;
  const rate = stats?.occupancy_rate || 0;

  const data = [
    { name: 'Occupé', value: occupied },
    { name: 'Disponible', value: available },
  ];

  const COLORS = ['#059669', '#e5e7eb'];

  return (
    <Card title="Taux d'occupation">
      <div className="h-64 relative">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={data}
              cx="50%"
              cy="50%"
              innerRadius={60}
              outerRadius={90}
              dataKey="value"
              startAngle={90}
              endAngle={-270}
            >
              {data.map((_, index) => (
                <Cell key={`cell-${index}`} fill={COLORS[index]} />
              ))}
            </Pie>
          </PieChart>
        </ResponsiveContainer>
        <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
          <span className="text-3xl font-bold text-gray-900">{rate}%</span>
          <span className="text-sm text-gray-500">Occupé</span>
        </div>
      </div>
      <div className="flex justify-center gap-6 mt-2">
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-emerald-600" />
          <span className="text-sm text-gray-600">Occupé ({occupied})</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-gray-200" />
          <span className="text-sm text-gray-600">Disponible ({available})</span>
        </div>
      </div>
    </Card>
  );
}
