import { useState, useCallback } from 'react';
import { useApi } from '../hooks/useApi';
import { getProperties, createProperty, updateProperty, deleteProperty } from '../api/properties';
import type { Property } from '../types';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import PropertyList from '../components/properties/PropertyList';
import PropertyForm from '../components/properties/PropertyForm';

export default function Properties() {
  const [showModal, setShowModal] = useState(false);
  const [editingProperty, setEditingProperty] = useState<Property | undefined>();

  const fetchProperties = useCallback(() => getProperties(), []);
  const { data: properties, loading, refetch } = useApi<Property[]>(fetchProperties);

  const handleCreate = () => {
    setEditingProperty(undefined);
    setShowModal(true);
  };

  const handleEdit = (property: Property) => {
    setEditingProperty(property);
    setShowModal(true);
  };

  const handleDelete = async (id: number) => {
    if (!window.confirm('Êtes-vous sûr de vouloir supprimer ce bien ?')) return;
    try {
      await deleteProperty(id);
      refetch();
    } catch {
      alert('Erreur lors de la suppression');
    }
  };

  const handleSubmit = async (data: Partial<Property>) => {
    try {
      if (editingProperty) {
        await updateProperty(editingProperty.id, data);
      } else {
        await createProperty(data);
      }
      setShowModal(false);
      refetch();
    } catch {
      alert("Erreur lors de l'enregistrement");
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Propriétés</h1>
        <Button onClick={handleCreate}>Ajouter un bien</Button>
      </div>

      {loading ? (
        <div className="text-center py-8 text-gray-500">Chargement...</div>
      ) : (
        <PropertyList
          properties={properties || []}
          onEdit={handleEdit}
          onDelete={handleDelete}
        />
      )}

      <Modal
        isOpen={showModal}
        onClose={() => setShowModal(false)}
        title={editingProperty ? 'Modifier le bien' : 'Ajouter un bien'}
      >
        <PropertyForm
          property={editingProperty}
          onSubmit={handleSubmit}
          onCancel={() => setShowModal(false)}
        />
      </Modal>
    </div>
  );
}
