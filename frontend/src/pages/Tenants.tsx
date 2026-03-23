import { useState, useCallback } from 'react';
import { useApi } from '../hooks/useApi';
import { getTenants, createTenant, updateTenant, deleteTenant } from '../api/tenants';
import type { Tenant } from '../types';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import TenantList from '../components/tenants/TenantList';
import TenantForm from '../components/tenants/TenantForm';

export default function Tenants() {
  const [showModal, setShowModal] = useState(false);
  const [editingTenant, setEditingTenant] = useState<Tenant | undefined>();

  const fetchTenants = useCallback(() => getTenants(), []);
  const { data: tenants, loading, refetch } = useApi<Tenant[]>(fetchTenants);

  const handleCreate = () => {
    setEditingTenant(undefined);
    setShowModal(true);
  };

  const handleEdit = (tenant: Tenant) => {
    setEditingTenant(tenant);
    setShowModal(true);
  };

  const handleDelete = async (id: number) => {
    if (!window.confirm('Êtes-vous sûr de vouloir supprimer ce locataire ?')) return;
    try {
      await deleteTenant(id);
      refetch();
    } catch {
      alert('Erreur lors de la suppression');
    }
  };

  const handleSubmit = async (data: Partial<Tenant>) => {
    try {
      if (editingTenant) {
        await updateTenant(editingTenant.id, data);
      } else {
        await createTenant(data);
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
        <h1 className="text-2xl font-bold text-gray-900">Locataires</h1>
        <Button onClick={handleCreate}>Ajouter un locataire</Button>
      </div>

      {loading ? (
        <div className="text-center py-8 text-gray-500">Chargement...</div>
      ) : (
        <TenantList
          tenants={tenants || []}
          onEdit={handleEdit}
          onDelete={handleDelete}
        />
      )}

      <Modal
        isOpen={showModal}
        onClose={() => setShowModal(false)}
        title={editingTenant ? 'Modifier le locataire' : 'Ajouter un locataire'}
      >
        <TenantForm
          tenant={editingTenant}
          onSubmit={handleSubmit}
          onCancel={() => setShowModal(false)}
        />
      </Modal>
    </div>
  );
}
