export interface User {
  id: number;
  email: string;
  role: string;
  first_name: string;
  last_name: string;
  phone: string;
  organization_id: number;
  is_active: boolean;
}

export interface Organization {
  id: number;
  name: string;
  org_type: string;
  plan: string;
  phone: string;
  email: string;
}

export interface Property {
  id: number;
  organization_id: number;
  reference: string;
  property_type: string;
  address: string;
  city: string;
  neighborhood: string;
  surface_m2: number;
  rooms: number;
  rent_amount: number;
  charges_amount: number;
  status: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface Tenant {
  id: number;
  organization_id: number;
  first_name: string;
  last_name: string;
  phone: string;
  email: string;
  cni_number: string;
  profession: string;
  employer: string;
  emergency_contact: string;
  emergency_phone: string;
  created_at: string;
}

export interface Lease {
  id: number;
  property_id: number;
  tenant_id: number;
  start_date: string;
  end_date: string;
  rent_amount: number;
  deposit_amount: number;
  payment_day: number;
  status: string;
  contract_pdf_url: string;
  signed_at: string;
  created_at: string;
  property_reference?: string;
  property_address?: string;
  tenant_name?: string;
}

export interface Payment {
  id: number;
  lease_id: number;
  amount: number;
  due_date: string;
  paid_date: string;
  payment_method: string;
  reference: string;
  status: string;
  created_at: string;
}

export interface Invoice {
  id: number;
  payment_id: number;
  invoice_number: string;
  pdf_url: string;
  sent_at: string;
  created_at: string;
}

export interface PaginationMeta {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  meta?: PaginationMeta;
}

export interface DashboardStats {
  total_properties: number;
  occupied_properties: number;
  total_tenants: number;
  monthly_revenue: number;
  overdue_count: number;
  occupancy_rate: number;
}
