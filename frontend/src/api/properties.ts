import client from './client';
import type { Property, ApiResponse } from '../types';

export async function getProperties(params?: Record<string, string | number>): Promise<ApiResponse<Property[]>> {
  const response = await client.get('/v1/properties', { params });
  return response.data;
}

export async function getProperty(id: number): Promise<Property> {
  const response = await client.get(`/v1/properties/${id}`);
  return response.data.data;
}

export async function createProperty(data: Partial<Property>): Promise<Property> {
  const response = await client.post('/v1/properties', data);
  return response.data.data;
}

export async function updateProperty(id: number, data: Partial<Property>): Promise<Property> {
  const response = await client.put(`/v1/properties/${id}`, data);
  return response.data.data;
}

export async function deleteProperty(id: number): Promise<void> {
  await client.delete(`/v1/properties/${id}`);
}
