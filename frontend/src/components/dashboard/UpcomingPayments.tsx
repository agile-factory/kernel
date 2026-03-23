import type { Payment } from '../../types';
import { formatCurrency } from '../../utils/formatCurrency';
import { formatShortDate } from '../../utils/formatDate';
import Card from '../ui/Card';
import { Calendar } from 'lucide-react';

interface UpcomingPaymentsProps {
  payments?: Payment[];
}

export default function UpcomingPayments({ payments }: UpcomingPaymentsProps) {
  const getDaysUntilDue = (dueDate: string) => {
    const due = new Date(dueDate).getTime();
    const now = Date.now();
    return Math.ceil((due - now) / (1000 * 60 * 60 * 24));
  };

  return (
    <Card title="Paiements à venir">
      {!payments || payments.length === 0 ? (
        <div className="text-center py-8 text-gray-500 text-sm">
          Aucun paiement à venir
        </div>
      ) : (
        <div className="divide-y divide-gray-100">
          {payments.map((payment) => {
            const daysUntil = getDaysUntilDue(payment.due_date);
            return (
              <div key={payment.id} className="flex items-center justify-between py-3">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-gray-50 rounded-lg">
                    <Calendar size={18} className="text-gray-500" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-gray-900">
                      Bail #{payment.lease_id}
                    </p>
                    <p className="text-xs text-gray-500">
                      Échéance : {formatShortDate(payment.due_date)}
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="text-sm font-semibold text-gray-900">
                    {formatCurrency(payment.amount)}
                  </p>
                  <p className={`text-xs ${daysUntil <= 3 ? 'text-amber-600' : 'text-gray-500'}`}>
                    {daysUntil > 0
                      ? `Dans ${daysUntil} jour${daysUntil > 1 ? 's' : ''}`
                      : daysUntil === 0
                      ? "Aujourd'hui"
                      : 'En retard'}
                  </p>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </Card>
  );
}
