import type { Payment } from '../../types';
import { formatCurrency } from '../../utils/formatCurrency';
import { formatShortDate } from '../../utils/formatDate';
import Table from '../ui/Table';
import Badge, { getStatusVariant } from '../ui/Badge';

const statusLabels: Record<string, string> = {
  paid: 'Payé',
  pending: 'En attente',
  late: 'En retard',
  overdue: 'Impayé',
};

const methodLabels: Record<string, string> = {
  cash: 'Espèces',
  orange_money: 'Orange Money',
  wave: 'Wave',
  bank_transfer: 'Virement bancaire',
  check: 'Chèque',
};

interface PaymentListProps {
  payments: Payment[];
  onEdit?: (payment: Payment) => void;
}

export default function PaymentList({ payments, onEdit }: PaymentListProps) {
  const columns = [
    {
      key: 'lease_id',
      label: 'Bail',
      render: (row: Payment) => `#${row.lease_id}`,
    },
    {
      key: 'amount',
      label: 'Montant',
      render: (row: Payment) => formatCurrency(row.amount),
    },
    {
      key: 'due_date',
      label: "Date d'échéance",
      render: (row: Payment) => formatShortDate(row.due_date),
    },
    {
      key: 'paid_date',
      label: 'Date de paiement',
      render: (row: Payment) => row.paid_date ? formatShortDate(row.paid_date) : '-',
    },
    {
      key: 'payment_method',
      label: 'Méthode',
      render: (row: Payment) => methodLabels[row.payment_method] || row.payment_method || '-',
    },
    {
      key: 'status',
      label: 'Statut',
      render: (row: Payment) => (
        <Badge variant={getStatusVariant(row.status)}>
          {statusLabels[row.status] || row.status}
        </Badge>
      ),
    },
  ];

  return <Table columns={columns} data={payments} onRowClick={onEdit} />;
}
