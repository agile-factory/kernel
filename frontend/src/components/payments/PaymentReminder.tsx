import type { Payment } from '../../types';
import { formatCurrency } from '../../utils/formatCurrency';
import { formatShortDate } from '../../utils/formatDate';
import { sendReminders } from '../../api/payments';
import Button from '../ui/Button';
import Card from '../ui/Card';
import { AlertTriangle, Send } from 'lucide-react';

interface PaymentReminderProps {
  overduePayments: Payment[];
}

export default function PaymentReminder({ overduePayments }: PaymentReminderProps) {
  const getDaysOverdue = (dueDate: string) => {
    const due = new Date(dueDate).getTime();
    const now = Date.now();
    return Math.max(0, Math.floor((now - due) / (1000 * 60 * 60 * 24)));
  };

  const handleSendReminders = async () => {
    try {
      await sendReminders();
      alert('Rappels envoyés avec succès');
    } catch {
      alert("Erreur lors de l'envoi des rappels");
    }
  };

  return (
    <Card title="Paiements en retard">
      <div className="space-y-3">
        {overduePayments.map((payment) => {
          const daysOverdue = getDaysOverdue(payment.due_date);
          return (
            <div
              key={payment.id}
              className="flex items-center justify-between p-3 bg-amber-50 border border-amber-200 rounded-lg"
            >
              <div className="flex items-center gap-3">
                <div className="p-2 bg-amber-100 rounded-lg">
                  <AlertTriangle size={18} className="text-amber-600" />
                </div>
                <div>
                  <p className="text-sm font-medium text-gray-900">
                    Bail #{payment.lease_id}
                  </p>
                  <p className="text-xs text-gray-500">
                    {formatCurrency(payment.amount)} - Échéance : {formatShortDate(payment.due_date)}
                  </p>
                </div>
              </div>
              <div className="text-right">
                <span className="inline-block px-2 py-1 bg-red-100 text-red-700 text-xs font-medium rounded-full">
                  {daysOverdue} jour{daysOverdue > 1 ? 's' : ''} de retard
                </span>
              </div>
            </div>
          );
        })}
      </div>
      <div className="mt-4 flex justify-end">
        <Button variant="secondary" size="sm" onClick={handleSendReminders}>
          <Send size={14} className="mr-2" />
          Envoyer les rappels
        </Button>
      </div>
    </Card>
  );
}
