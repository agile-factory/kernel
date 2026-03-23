import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import Card from '../ui/Card';

interface RevenueChartProps {
  data?: { month: string; revenue: number }[];
}

const mockData = [
  { month: 'Jan', revenue: 4200000 },
  { month: 'Fév', revenue: 4500000 },
  { month: 'Mar', revenue: 4300000 },
  { month: 'Avr', revenue: 4800000 },
  { month: 'Mai', revenue: 4600000 },
  { month: 'Jun', revenue: 5100000 },
  { month: 'Jul', revenue: 4900000 },
  { month: 'Aoû', revenue: 5200000 },
  { month: 'Sep', revenue: 5000000 },
  { month: 'Oct', revenue: 5400000 },
  { month: 'Nov', revenue: 5100000 },
  { month: 'Déc', revenue: 5500000 },
];

const formatFCFA = (value: number) => {
  if (value >= 1000000) return `${(value / 1000000).toFixed(1)}M`;
  if (value >= 1000) return `${(value / 1000).toFixed(0)}K`;
  return String(value);
};

export default function RevenueChart({ data }: RevenueChartProps) {
  const chartData = data || mockData;

  return (
    <Card title="Revenus mensuels">
      <div className="h-64">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" vertical={false} />
            <XAxis dataKey="month" tick={{ fontSize: 12 }} />
            <YAxis tickFormatter={formatFCFA} tick={{ fontSize: 12 }} />
            <Tooltip
              formatter={(value: number) => [`${new Intl.NumberFormat('fr-FR').format(value)} FCFA`, 'Revenus']}
              contentStyle={{ borderRadius: '8px', border: '1px solid #e5e7eb' }}
            />
            <Bar dataKey="revenue" fill="#059669" radius={[4, 4, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </Card>
  );
}
