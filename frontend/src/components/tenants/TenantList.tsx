import type { Tenant } from '../../types';
import Table from '../ui/Table';
import { Pencil, Trash2 } from 'lucide-react';

interface TenantListProps {
  tenants: Tenant[];
  onEdit: (tenant: Tenant) => void;
  onDelete: (id: number) => void;
}

export default function TenantList({ tenants, onEdit, onDelete }: TenantListProps) {
  const columns = [
    {
      key: 'name',
      label: 'Nom complet',
      render: (row: Tenant) => `${row.first_name} ${row.last_name}`,
    },
    { key: 'phone', label: 'Téléphone' },
    { key: 'email', label: 'Email' },
    { key: 'cni_number', label: 'CNI' },
    { key: 'profession', label: 'Profession' },
    {
      key: 'actions',
      label: 'Actions',
      render: (row: Tenant) => (
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

  return <Table columns={columns} data={tenants} />;
}
