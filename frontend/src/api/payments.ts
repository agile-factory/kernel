import client from './client';
import type { Payment, ApiResponse } from '../types';

export async function getPayments(params?: Record<string, string>) {
  const { data } = await client.get<ApiResponse<Payment[]>>('/v1/payments', { params });
  return data;
}

export async function getPayment(id: string) {
  const { data } = await client.get<ApiResponse<Payment>>(`/v1/payments/${id}`);
  return data;
}

export async function createPayment(paymentData: Partial<Payment>) {
  const { data } = await client.post<ApiResponse<Payment>>('/v1/payments', paymentData);
  return data;
}

export async function getOverduePayments() {
  const { data } = await client.get<ApiResponse<Payment[]>>('/v1/payments/overdue');
  return data;
}

export async function sendReminders() {
  const { data } = await client.post<ApiResponse<void>>('/v1/payments/reminders');
  return data;
}
