import { useState } from 'react';
import type { Payment, Lease } from '../../types';
import Input from '../ui/Input';
import Button from '../ui/Button';

interface PaymentFormProps {
  payment?: Payment;
  leases: Lease[];
  onSubmit: (data: Partial<Payment>) => void;
  onCancel: () => void;
}

export default function PaymentForm({ payment, leases, onSubmit, onCancel }: PaymentFormProps) {
  const [form, setForm] = useState({
    lease_id: payment?.lease_id || '',
    amount: payment?.amount || 0,
    payment_method: payment?.payment_method || 'cash',
    reference: payment?.reference || '',
    paid_date: payment?.paid_date?.slice(0, 10) || new Date().toISOString().slice(0, 10),
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
      lease_id: Number(form.lease_id),
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-1">
        <label className="block text-sm font-medium text-gray-700">Bail *</label>
        <select
          name="lease_id"
          value={form.lease_id}
          onChange={handleChange}
          required
          className="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
        >
          <option value="">Sélectionner un bail</option>
          {leases.map((l) => (
            <option key={l.id} value={l.id}>
              {l.property_reference || `Propriété #${l.property_id}`} - {l.tenant_name || `Locataire #${l.tenant_id}`}
            </option>
          ))}
        </select>
      </div>
      <Input
        label="Montant (FCFA)"
        type="number"
        name="amount"
        value={form.amount}
        onChange={handleChange}
        required
      />
      <div className="space-y-1">
        <label className="block text-sm font-medium text-gray-700">Méthode de paiement *</label>
        <select
          name="payment_method"
          value={form.payment_method}
          onChange={handleChange}
          required
          className="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
        >
          <option value="cash">Espèces</option>
          <option value="orange_money">Orange Money</option>
          <option value="wave">Wave</option>
          <option value="bank_transfer">Virement bancaire</option>
          <option value="check">Chèque</option>
        </select>
      </div>
      <Input
        label="Référence"
        name="reference"
        value={form.reference}
        onChange={handleChange}
        placeholder="Numéro de transaction"
      />
      <Input
        label="Date de paiement"
        type="date"
        name="paid_date"
        value={form.paid_date}
        onChange={handleChange}
        required
      />
      <div className="flex justify-end gap-3 pt-4">
        <Button variant="ghost" onClick={onCancel}>
          Annuler
        </Button>
        <Button type="submit">
          {payment ? 'Modifier' : 'Enregistrer'}
        </Button>
      </div>
    </form>
  );
}
