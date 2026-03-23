import { useState, useCallback } from 'react';
import { useApi } from '../hooks/useApi';
import { getPayments, createPayment, getOverduePayments } from '../api/payments';
import { getLeases } from '../api/leases';
import type { Payment, Lease } from '../types';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import PaymentList from '../components/payments/PaymentList';
import PaymentForm from '../components/payments/PaymentForm';
import PaymentReminder from '../components/payments/PaymentReminder';

export default function Payments() {
  const [showModal, setShowModal] = useState(false);

  const fetchPayments = useCallback(() => getPayments(), []);
  const fetchLeases = useCallback(() => getLeases(), []);
  const fetchOverdue = useCallback(() => getOverduePayments(), []);

  const { data: payments, loading, refetch } = useApi<Payment[]>(fetchPayments);
  const { data: leases } = useApi<Lease[]>(fetchLeases);
  const { data: overduePayments, refetch: refetchOverdue } = useApi<Payment[]>(fetchOverdue);

  const handleCreate = () => {
    setShowModal(true);
  };

  const handleSubmit = async (data: Partial<Payment>) => {
    try {
      await createPayment(data);
      setShowModal(false);
      refetch();
      refetchOverdue();
    } catch {
      alert("Erreur lors de l'enregistrement du paiement");
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Paiements</h1>
        <Button onClick={handleCreate}>Enregistrer un paiement</Button>
      </div>

      {overduePayments && overduePayments.length > 0 && (
        <PaymentReminder overduePayments={overduePayments} />
      )}

      {loading ? (
        <div className="text-center py-8 text-gray-500">Chargement...</div>
      ) : (
        <PaymentList payments={payments || []} />
      )}

      <Modal
        isOpen={showModal}
        onClose={() => setShowModal(false)}
        title="Enregistrer un paiement"
      >
        <PaymentForm
          leases={leases || []}
          onSubmit={handleSubmit}
          onCancel={() => setShowModal(false)}
        />
      </Modal>
    </div>
  );
}
