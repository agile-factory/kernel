import type { Property } from '../../types';
import { formatCurrency } from '../../utils/formatCurrency';
import Badge, { getStatusVariant } from '../ui/Badge';
import { MapPin, Home, Maximize } from 'lucide-react';

const typeLabels: Record<string, string> = {
  apartment: 'Appartement',
  house: 'Maison',
  land: 'Terrain',
  commercial: 'Commercial',
  studio: 'Studio',
  villa: 'Villa',
};

const statusLabels: Record<string, string> = {
  available: 'Disponible',
  rented: 'Loué',
  maintenance: 'En maintenance',
};

interface PropertyCardProps {
  property: Property;
}

export default function PropertyCard({ property }: PropertyCardProps) {
  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden hover:shadow-md transition-shadow">
      <div className="p-5">
        <div className="flex items-center justify-between mb-3">
          <Badge variant={getStatusVariant(property.property_type === 'villa' ? 'info' : 'neutral')}>
            {typeLabels[property.property_type] || property.property_type}
          </Badge>
          <Badge variant={getStatusVariant(property.status)}>
            {statusLabels[property.status] || property.status}
          </Badge>
        </div>
        <h3 className="text-lg font-semibold text-gray-900 mb-1">{property.reference}</h3>
        <div className="flex items-center gap-1 text-sm text-gray-500 mb-3">
          <MapPin size={14} />
          <span>{property.address}, {property.city}</span>
        </div>
        <div className="flex items-center gap-4 text-sm text-gray-600 mb-4">
          {property.rooms > 0 && (
            <div className="flex items-center gap-1">
              <Home size={14} />
              <span>{property.rooms} pièce{property.rooms > 1 ? 's' : ''}</span>
            </div>
          )}
          {property.surface_m2 > 0 && (
            <div className="flex items-center gap-1">
              <Maximize size={14} />
              <span>{property.surface_m2} m²</span>
            </div>
          )}
        </div>
        <div className="text-lg font-bold text-emerald-600">
          {formatCurrency(property.rent_amount)}
          <span className="text-sm font-normal text-gray-500"> /mois</span>
        </div>
      </div>
    </div>
  );
}
