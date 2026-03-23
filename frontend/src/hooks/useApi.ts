import { useState, useEffect, useCallback } from 'react';
import type { ApiResponse } from '../types';

export function useApi<T>(fetchFn: () => Promise<ApiResponse<T>>) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refetch = useCallback(() => {
    setLoading(true);
    setError(null);
    fetchFn()
      .then((response) => {
        if (response.success && response.data !== undefined) {
          setData(response.data);
        } else {
          setError(response.error || 'Une erreur est survenue');
        }
      })
      .catch((err) => {
        setError(err.response?.data?.error || err.message || 'Une erreur est survenue');
      })
      .finally(() => {
        setLoading(false);
      });
  }, [fetchFn]);

  useEffect(() => {
    refetch();
  }, [refetch]);

  return { data, loading, error, refetch };
}
