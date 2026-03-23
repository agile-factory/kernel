import { useCallback } from 'react';
import { useApi } from '../hooks/useApi';
import type { DashboardStats, Payment } from '../types';
import StatsCards from '../components/dashboard/StatsCards';
import RevenueChart from '../components/dashboard/RevenueChart';
import OccupancyRate from '../components/dashboard/OccupancyRate';
import UpcomingPayments from '../components/dashboard/UpcomingPayments';
import client from '../api/client';

export default function Dashboard() {
  const fetchStats = useCallback(
    () => client.get('/v1/dashboard/stats').then((r) => r.data),
    []
  );
  const fetchUpcoming = useCallback(
    () => client.get('/v1/payments?status=pending&limit=5').then((r) => r.data),
    []
  );

  const { data: stats, loading: loadingStats } = useApi<DashboardStats>(fetchStats);
  const { data: upcomingPayments } = useApi<Payment[]>(fetchUpcoming);

  if (loadingStats) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500">Chargement...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-gray-900">Tableau de bord</h1>

      {stats && <StatsCards stats={stats} />}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <RevenueChart />
        {stats && <OccupancyRate stats={stats} />}
      </div>

      <UpcomingPayments payments={upcomingPayments || undefined} />
    </div>
  );
}
