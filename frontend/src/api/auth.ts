import client from './client';
import type { User } from '../types';

export interface AuthResponse {
  token: string;
  user: User;
}

export interface RegisterData {
  organization_name: string;
  organization_type: string;
  first_name: string;
  last_name: string;
  email: string;
  phone: string;
  password: string;
}

export async function login(email: string, password: string): Promise<AuthResponse> {
  const response = await client.post('/v1/auth/login', { email, password });
  return response.data.data;
}

export async function register(data: RegisterData): Promise<AuthResponse> {
  const response = await client.post('/v1/auth/register', data);
  return response.data.data;
}

export async function refreshToken(): Promise<{ token: string }> {
  const response = await client.post('/v1/auth/refresh');
  return response.data.data;
}

export async function getMe(): Promise<User> {
  const response = await client.get('/v1/auth/me');
  return response.data.data;
}
