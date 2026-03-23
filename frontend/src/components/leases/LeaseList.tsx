import type { Lease } from '../../types';
import { formatCurrency } from '../../utils/formatCurrency';
import { formatShortDate } from '../../utils/formatDate';
import Table from '../ui/Table';
import Badge, { getStatusVariant } from '../ui/Badge';
import { Pencil, XCircle } from 'lucide-react';

const statusLabels: Record<string, string> = {
  active: 'Actif',
  terminated: 'Résilié',
  pending: 'En attente',
  expired: 'Expiré',
};

interface LeaseListProps {
  leases: Lease[];
  onEdit: (lease: Lease) => void;
  onTerminate: (id: number) => void;
}

export default function LeaseList({ leases, onEdit, onTerminate }: LeaseListProps) {
  const columns = [
    {
      key: 'property_reference',
      label: 'Propriété',
      render: (row: Lease) => row.property_reference || `#${row.property_id}`,
    },
    {
      key: 'tenant_name',
      label: 'Locataire',
      render: (row: Lease) => row.tenant_name || `#${row.tenant_id}`,
    },
    {
      key: 'start_date',
      label: 'Début',
      render: (row: Lease) => formatShortDate(row.start_date),
    },
    {
      key: 'end_date',
      label: 'Fin',
      render: (row: Lease) => formatShortDate(row.end_date),
    },
    {
      key: 'rent_amount',
      label: 'Loyer',
      render: (row: Lease) => formatCurrency(row.rent_amount),
    },
    {
      key: 'status',
      label: 'Statut',
      render: (row: Lease) => (
        <Badge variant={getStatusVariant(row.status)}>
          {statusLabels[row.status] || row.status}
        </Badge>
      ),
    },
    {
      key: 'actions',
      label: 'Actions',
      render: (row: Lease) => (
        <div className="flex items-center gap-2">
          <button
            onClick={(e) => { e.stopPropagation(); onEdit(row); }}
            className="p-1.5 rounded-lg text-gray-500 hover:bg-gray-100 hover:text-emerald-600 transition-colors"
            title="Modifier"
          >
            <Pencil size={16} />
          </button>
          {row.status === 'active' && (
            <button
              onClick={(e) => { e.stopPropagation(); onTerminate(row.id); }}
              className="p-1.5 rounded-lg text-gray-500 hover:bg-gray-100 hover:text-red-600 transition-colors"
              title="Résilier"
            >
              <XCircle size={16} />
            </button>
          )}
        </div>
      ),
    },
  ];

  return <Table columns={columns} data={leases} />;
}
