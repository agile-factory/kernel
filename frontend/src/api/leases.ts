import client from './client';
import type { Lease, ApiResponse } from '../types';

export async function getLeases(params?: Record<string, string>) {
  const { data } = await client.get<ApiResponse<Lease[]>>('/v1/leases', { params });
  return data;
}

export async function getLease(id: string) {
  const { data } = await client.get<ApiResponse<Lease>>(`/v1/leases/${id}`);
  return data;
}

export async function createLease(leaseData: Partial<Lease>) {
  const { data } = await client.post<ApiResponse<Lease>>('/v1/leases', leaseData);
  return data;
}

export async function updateLease(id: string, leaseData: Partial<Lease>) {
  const { data } = await client.put<ApiResponse<Lease>>(`/v1/leases/${id}`, leaseData);
  return data;
}

export async function terminateLease(id: string) {
  const { data } = await client.post<ApiResponse<Lease>>(`/v1/leases/${id}/terminate`);
  return data;
}
