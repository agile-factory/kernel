import type { Property } from '../../types';
import { formatCurrency } from '../../utils/formatCurrency';
import Table from '../ui/Table';
import Badge, { getStatusVariant } from '../ui/Badge';
import Button from '../ui/Button';
import { Pencil, Trash2 } from 'lucide-react';

const statusLabels: Record<string, string> = {
  available: 'Disponible',
  rented: 'Loué',
  maintenance: 'En maintenance',
};

interface PropertyListProps {
  properties: Property[];
  onEdit: (property: Property) => void;
  onDelete: (id: number) => void;
}

export default function PropertyList({ properties, onEdit, onDelete }: PropertyListProps) {
  const columns = [
    { key: 'reference', label: 'Référence' },
    { key: 'property_type', label: 'Type' },
    { key: 'address', label: 'Adresse' },
    { key: 'city', label: 'Ville' },
    {
      key: 'rent_amount',
      label: 'Loyer',
      render: (row: Property) => formatCurrency(row.rent_amount),
    },
    {
      key: 'status',
      label: 'Statut',
      render: (row: Property) => (
        <Badge variant={getStatusVariant(row.status)}>
          {statusLabels[row.status] || row.status}
        </Badge>
      ),
    },
    {
      key: 'actions',
      label: 'Actions',
      render: (row: Property) => (
        <div className="flex items-center gap-2">
          <button
            onClick={(e) => { e.stopPropagation(); onEdit(row); }}
            className="p-1.5 rounded-lg text-gray-500 hover:bg-gray-100 hover:text-emerald-600 transition-colors"
            title="Modifier"
          >
            <Pencil size={16} />
          </button>
          <button
            onClick={(e) => { e.stopPropagation(); onDelete(row.id); }}
            className="p-1.5 rounded-lg text-gray-500 hover:bg-gray-100 hover:text-red-600 transition-colors"
            title="Supprimer"
          >
            <Trash2 size={16} />
          </button>
        </div>
      ),
    },
  ];

  return <Table columns={columns} data={properties} />;
}
