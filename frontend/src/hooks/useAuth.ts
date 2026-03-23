import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../store/authStore';
import { login as apiLogin, register as apiRegister, getMe, type RegisterData } from '../api/auth';

export function useAuth() {
  const navigate = useNavigate();
  const { user, isAuthenticated, setAuth, setUser, logout: storeLogout } = useAuthStore();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const token = useAuthStore.getState().token;
    if (token && !user) {
      setLoading(true);
      getMe()
        .then((userData) => {
          setUser(userData);
        })
        .catch(() => {
          storeLogout();
        })
        .finally(() => {
          setLoading(false);
        });
    }
  }, []);

  const login = async (email: string, password: string) => {
    setLoading(true);
    try {
      const { user: userData, token } = await apiLogin(email, password);
      setAuth(userData, token);
      navigate('/dashboard');
    } finally {
      setLoading(false);
    }
  };

  const register = async (data: RegisterData) => {
    setLoading(true);
    try {
      const { user: userData, token } = await apiRegister(data);
      setAuth(userData, token);
      navigate('/dashboard');
    } finally {
      setLoading(false);
    }
  };

  const logout = () => {
    storeLogout();
    navigate('/login');
  };

  return { user, isAuthenticated, login, logout, register, loading };
}
