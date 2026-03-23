import { useState, useCallback } from 'react';
import { useApi } from '../hooks/useApi';
import { getLeases, createLease, updateLease, terminateLease } from '../api/leases';
import { getProperties } from '../api/properties';
import { getTenants } from '../api/tenants';
import type { Lease, Property, Tenant } from '../types';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import LeaseList from '../components/leases/LeaseList';
import LeaseForm from '../components/leases/LeaseForm';

export default function Leases() {
  const [showModal, setShowModal] = useState(false);
  const [editingLease, setEditingLease] = useState<Lease | undefined>();

  const fetchLeases = useCallback(() => getLeases(), []);
  const fetchProperties = useCallback(() => getProperties(), []);
  const fetchTenants = useCallback(() => getTenants(), []);

  const { data: leases, loading, refetch } = useApi<Lease[]>(fetchLeases);
  const { data: properties } = useApi<Property[]>(fetchProperties);
  const { data: tenants } = useApi<Tenant[]>(fetchTenants);

  const handleCreate = () => {
    setEditingLease(undefined);
    setShowModal(true);
  };

  const handleEdit = (lease: Lease) => {
    setEditingLease(lease);
    setShowModal(true);
  };

  const handleTerminate = async (id: number) => {
    if (!window.confirm('Êtes-vous sûr de vouloir résilier ce bail ?')) return;
    try {
      await terminateLease(String(id));
      refetch();
    } catch {
      alert('Erreur lors de la résiliation');
    }
  };

  const handleSubmit = async (data: Partial<Lease>) => {
    try {
      if (editingLease) {
        await updateLease(String(editingLease.id), data);
      } else {
        await createLease(data);
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
        <h1 className="text-2xl font-bold text-gray-900">Baux</h1>
        <Button onClick={handleCreate}>Nouveau bail</Button>
      </div>

      {loading ? (
        <div className="text-center py-8 text-gray-500">Chargement...</div>
      ) : (
        <LeaseList
          leases={leases || []}
          onEdit={handleEdit}
          onTerminate={handleTerminate}
        />
      )}

      <Modal
        isOpen={showModal}
        onClose={() => setShowModal(false)}
        title={editingLease ? 'Modifier le bail' : 'Nouveau bail'}
      >
        <LeaseForm
          lease={editingLease}
          properties={properties || []}
          tenants={tenants || []}
          onSubmit={handleSubmit}
          onCancel={() => setShowModal(false)}
        />
      </Modal>
    </div>
  );
}
