import { useState } from 'react';
import type { Lease, Property, Tenant } from '../../types';
import Input from '../ui/Input';
import Button from '../ui/Button';

interface LeaseFormProps {
  lease?: Lease;
  properties: Property[];
  tenants: Tenant[];
  onSubmit: (data: Partial<Lease>) => void;
  onCancel: () => void;
}

export default function LeaseForm({ lease, properties, tenants, onSubmit, onCancel }: LeaseFormProps) {
  const [form, setForm] = useState({
    property_id: lease?.property_id || '',
    tenant_id: lease?.tenant_id || '',
    start_date: lease?.start_date?.slice(0, 10) || '',
    end_date: lease?.end_date?.slice(0, 10) || '',
    rent_amount: lease?.rent_amount || 0,
    deposit_amount: lease?.deposit_amount || 0,
    payment_day: lease?.payment_day || 5,
  });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value, type } = e.target;
    setForm({
      ...form,
      [name]: type === 'number' ? Number(value) : value,
    });
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({
      ...form,
      property_id: Number(form.property_id),
      tenant_id: Number(form.tenant_id),
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-1">
        <label className="block text-sm font-medium text-gray-700">Propriété *</label>
        <select
          name="property_id"
          value={form.property_id}
          onChange={handleChange}
          required
          className="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
        >
          <option value="">Sélectionner une propriété</option>
          {properties.map((p) => (
            <option key={p.id} value={p.id}>
              {p.reference} - {p.address}, {p.city}
            </option>
          ))}
        </select>
      </div>
      <div className="space-y-1">
        <label className="block text-sm font-medium text-gray-700">Locataire *</label>
        <select
          name="tenant_id"
          value={form.tenant_id}
          onChange={handleChange}
          required
          className="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
        >
          <option value="">Sélectionner un locataire</option>
          {tenants.map((t) => (
            <option key={t.id} value={t.id}>
              {t.first_name} {t.last_name}
            </option>
          ))}
        </select>
      </div>
      <div className="grid grid-cols-2 gap-4">
        <Input
          label="Date de début"
          type="date"
          name="start_date"
          value={form.start_date}
          onChange={handleChange}
          required
        />
        <Input
          label="Date de fin"
          type="date"
          name="end_date"
          value={form.end_date}
          onChange={handleChange}
          required
        />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <Input
          label="Loyer (FCFA)"
          type="number"
          name="rent_amount"
          value={form.rent_amount}
          onChange={handleChange}
          required
        />
        <Input
          label="Caution (FCFA)"
          type="number"
          name="deposit_amount"
          value={form.deposit_amount}
          onChange={handleChange}
          required
        />
      </div>
      <Input
        label="Jour de paiement (1-28)"
        type="number"
        name="payment_day"
        value={form.payment_day}
        onChange={handleChange}
        required
      />
      <div className="flex justify-end gap-3 pt-4">
        <Button variant="ghost" onClick={onCancel}>
          Annuler
        </Button>
        <Button type="submit">
          {lease ? 'Modifier' : 'Créer'}
        </Button>
      </div>
    </form>
  );
}
