import client from './client';
import type { Tenant, ApiResponse } from '../types';

export async function getTenants(params?: Record<string, string | number>): Promise<ApiResponse<Tenant[]>> {
  const response = await client.get('/v1/tenants', { params });
  return response.data;
}

export async function getTenant(id: number): Promise<Tenant> {
  const response = await client.get(`/v1/tenants/${id}`);
  return response.data.data;
}

export async function createTenant(data: Partial<Tenant>): Promise<Tenant> {
  const response = await client.post('/v1/tenants', data);
  return response.data.data;
}

export async function updateTenant(id: number, data: Partial<Tenant>): Promise<Tenant> {
  const response = await client.put(`/v1/tenants/${id}`, data);
  return response.data.data;
}

export async function deleteTenant(id: number): Promise<void> {
  await client.delete(`/v1/tenants/${id}`);
}
