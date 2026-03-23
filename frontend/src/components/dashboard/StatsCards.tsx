import type { DashboardStats } from '../../types';
import { formatCurrency } from '../../utils/formatCurrency';
import { Building2, Users, TrendingUp, AlertCircle } from 'lucide-react';

interface StatsCardsProps {
  stats: DashboardStats;
}

export default function StatsCards({ stats }: StatsCardsProps) {
  const cards = [
    {
      label: 'Biens totaux',
      value: stats.total_properties,
      icon: Building2,
      color: 'text-emerald-600',
      bg: 'bg-emerald-50',
    },
    {
      label: "Taux d'occupation",
      value: `${stats.occupancy_rate}%`,
      icon: Users,
      color: 'text-blue-600',
      bg: 'bg-blue-50',
    },
    {
      label: 'Revenus du mois',
      value: formatCurrency(stats.monthly_revenue),
      icon: TrendingUp,
      color: 'text-emerald-600',
      bg: 'bg-emerald-50',
    },
    {
      label: 'Impayés',
      value: stats.overdue_count,
      icon: AlertCircle,
      color: stats.overdue_count > 0 ? 'text-red-600' : 'text-amber-600',
      bg: stats.overdue_count > 0 ? 'bg-red-50' : 'bg-amber-50',
    },
  ];

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      {cards.map((card) => (
        <div
          key={card.label}
          className="bg-white rounded-xl shadow-sm border border-gray-200 p-5"
        >
          <div className="flex items-center justify-between mb-3">
            <span className="text-sm font-medium text-gray-500">{card.label}</span>
            <div className={`p-2 rounded-lg ${card.bg}`}>
              <card.icon size={20} className={card.color} />
            </div>
          </div>
          <p className="text-2xl font-bold text-gray-900">{card.value}</p>
        </div>
      ))}
    </div>
  );
}
