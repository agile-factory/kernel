import React from 'react';

interface BadgeProps {
  variant?: 'success' | 'warning' | 'danger' | 'info' | 'neutral';
  children: React.ReactNode;
}

const variantClasses: Record<string, string> = {
  success: 'bg-green-100 text-green-800',
  warning: 'bg-amber-100 text-amber-800',
  danger: 'bg-red-100 text-red-800',
  info: 'bg-blue-100 text-blue-800',
  neutral: 'bg-gray-100 text-gray-800',
};

const statusVariantMap: Record<string, string> = {
  available: 'success',
  active: 'success',
  paid: 'success',
  rented: 'info',
  pending: 'info',
  maintenance: 'warning',
  late: 'warning',
  overdue: 'warning',
  terminated: 'danger',
  cancelled: 'danger',
};

export function getStatusVariant(status: string): 'success' | 'warning' | 'danger' | 'info' | 'neutral' {
  return (statusVariantMap[status] || 'neutral') as any;
}

export default function Badge({ variant = 'neutral', children }: BadgeProps) {
  return (
    <span
      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${variantClasses[variant]}`}
    >
      {children}
    </span>
  );
}
