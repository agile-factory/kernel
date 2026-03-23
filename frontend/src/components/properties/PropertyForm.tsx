import { useState } from 'react';
import type { Property } from '../../types';
import Input from '../ui/Input';
import Button from '../ui/Button';

interface PropertyFormProps {
  property?: Property;
  onSubmit: (data: Partial<Property>) => void;
  onCancel: () => void;
}

export default function PropertyForm({ property, onSubmit, onCancel }: PropertyFormProps) {
  const [form, setForm] = useState({
    reference: property?.reference || '',
    property_type: property?.property_type || 'apartment',
    address: property?.address || '',
    city: property?.city || '',
    neighborhood: property?.neighborhood || '',
    surface_m2: property?.surface_m2 || 0,
    rooms: property?.rooms || 0,
    rent_amount: property?.rent_amount || 0,
    charges_amount: property?.charges_amount || 0,
    status: property?.status || 'available',
    description: property?.description || '',
  });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => {
    const { name, value, type } = e.target;
    setForm({
      ...form,
      [name]: type === 'number' ? Number(value) : value,
    });
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(form);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <Input
        label="Référence"
        name="reference"
        value={form.reference}
        onChange={handleChange}
        required
      />
      <div className="space-y-1">
        <label className="block text-sm font-medium text-gray-700">Type de bien *</label>
        <select
          name="property_type"
          value={form.property_type}
          onChange={handleChange}
          required
          className="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
        >
          <option value="apartment">Appartement</option>
          <option value="house">Maison</option>
          <option value="land">Terrain</option>
          <option value="commercial">Commercial</option>
          <option value="studio">Studio</option>
          <option value="villa">Villa</option>
        </select>
      </div>
      <Input
        label="Adresse"
        name="address"
        value={form.address}
        onChange={handleChange}
        required
      />
      <div className="grid grid-cols-2 gap-4">
        <Input
          label="Ville"
          name="city"
          value={form.city}
          onChange={handleChange}
          required
        />
        <Input
          label="Quartier"
          name="neighborhood"
          value={form.neighborhood}
          onChange={handleChange}
        />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <Input
          label="Surface (m²)"
          name="surface_m2"
          type="number"
          value={form.surface_m2}
          onChange={handleChange}
        />
        <Input
          label="Pièces"
          name="rooms"
          type="number"
          value={form.rooms}
          onChange={handleChange}
        />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <Input
          label="Loyer (FCFA)"
          name="rent_amount"
          type="number"
          value={form.rent_amount}
          onChange={handleChange}
          required
        />
        <Input
          label="Charges (FCFA)"
          name="charges_amount"
          type="number"
          value={form.charges_amount}
          onChange={handleChange}
        />
      </div>
      <div className="space-y-1">
        <label className="block text-sm font-medium text-gray-700">Statut</label>
        <select
          name="status"
          value={form.status}
          onChange={handleChange}
          className="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
        >
          <option value="available">Disponible</option>
          <option value="rented">Loué</option>
          <option value="maintenance">En maintenance</option>
        </select>
      </div>
      <div className="space-y-1">
        <label className="block text-sm font-medium text-gray-700">Description</label>
        <textarea
          name="description"
          value={form.description}
          onChange={handleChange}
          rows={3}
          className="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
        />
      </div>
      <div className="flex justify-end gap-3 pt-4">
        <Button variant="ghost" onClick={onCancel}>
          Annuler
        </Button>
        <Button type="submit">
          {property ? 'Modifier' : 'Ajouter'}
        </Button>
      </div>
    </form>
  );
}
