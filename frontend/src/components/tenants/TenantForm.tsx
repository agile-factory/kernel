import { useState } from 'react';
import type { Tenant } from '../../types';
import Input from '../ui/Input';
import Button from '../ui/Button';

interface TenantFormProps {
  tenant?: Tenant;
  onSubmit: (data: Partial<Tenant>) => void;
  onCancel: () => void;
}

export default function TenantForm({ tenant, onSubmit, onCancel }: TenantFormProps) {
  const [form, setForm] = useState({
    first_name: tenant?.first_name || '',
    last_name: tenant?.last_name || '',
    phone: tenant?.phone || '',
    email: tenant?.email || '',
    cni_number: tenant?.cni_number || '',
    profession: tenant?.profession || '',
    employer: tenant?.employer || '',
    emergency_contact: tenant?.emergency_contact || '',
    emergency_phone: tenant?.emergency_phone || '',
  });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setForm({ ...form, [e.target.name]: e.target.value });
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(form);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <Input
          label="Prénom"
          name="first_name"
          value={form.first_name}
          onChange={handleChange}
          required
        />
        <Input
          label="Nom"
          name="last_name"
          value={form.last_name}
          onChange={handleChange}
          required
        />
      </div>
      <Input
        label="Téléphone"
        name="phone"
        placeholder="+221 77 000 00 00"
        value={form.phone}
        onChange={handleChange}
        required
      />
      <Input
        label="Email"
        type="email"
        name="email"
        value={form.email}
        onChange={handleChange}
      />
      <Input
        label="Numéro CNI"
        name="cni_number"
        value={form.cni_number}
        onChange={handleChange}
        required
      />
      <div className="grid grid-cols-2 gap-4">
        <Input
          label="Profession"
          name="profession"
          value={form.profession}
          onChange={handleChange}
        />
        <Input
          label="Employeur"
          name="employer"
          value={form.employer}
          onChange={handleChange}
        />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <Input
          label="Contact d'urgence"
          name="emergency_contact"
          value={form.emergency_contact}
          onChange={handleChange}
        />
        <Input
          label="Tél. urgence"
          name="emergency_phone"
          value={form.emergency_phone}
          onChange={handleChange}
        />
      </div>
      <div className="flex justify-end gap-3 pt-4">
        <Button variant="ghost" onClick={onCancel}>
          Annuler
        </Button>
        <Button type="submit">
          {tenant ? 'Modifier' : 'Ajouter'}
        </Button>
      </div>
    </form>
  );
}
