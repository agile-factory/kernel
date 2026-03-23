import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import Input from '../components/ui/Input';
import Button from '../components/ui/Button';

export default function Register() {
  const { register, loading } = useAuth();
  const [error, setError] = useState('');
  const [form, setForm] = useState({
    organization_name: '',
    organization_type: 'agency',
    first_name: '',
    last_name: '',
    email: '',
    phone: '',
    password: '',
    confirmPassword: '',
  });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    setForm({ ...form, [e.target.name]: e.target.value });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (form.password !== form.confirmPassword) {
      setError('Les mots de passe ne correspondent pas');
      return;
    }
    try {
      const { confirmPassword, ...data } = form;
      await register(data);
    } catch (err: any) {
      setError(err.response?.data?.error || "Erreur lors de l'inscription");
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100 px-4 py-8">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-emerald-600">Lokatik</h1>
          <p className="text-gray-500 mt-2">Créez votre compte</p>
        </div>
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-8">
          <h2 className="text-xl font-semibold text-gray-900 mb-6">S'inscrire</h2>
          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg">
              {error}
            </div>
          )}
          <form onSubmit={handleSubmit} className="space-y-4">
            <Input
              label="Nom de l'organisation"
              name="organization_name"
              value={form.organization_name}
              onChange={handleChange}
              required
            />
            <div className="space-y-1">
              <label className="block text-sm font-medium text-gray-700">
                Type <span className="text-red-500 ml-1">*</span>
              </label>
              <select
                name="organization_type"
                value={form.organization_type}
                onChange={handleChange}
                required
                className="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
              >
                <option value="agency">Agence</option>
                <option value="owner">Propriétaire</option>
              </select>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <Input
                label="Prénom"
                name="first_name"
                value={form.first_name}
                onChange={handleChange}
                required
              />
              <Input
                label="Nom"
                name="last_name"
                value={form.last_name}
                onChange={handleChange}
                required
              />
            </div>
            <Input
              label="Email"
              type="email"
              name="email"
              value={form.email}
              onChange={handleChange}
              required
            />
            <Input
              label="Téléphone"
              name="phone"
              placeholder="+221 77 000 00 00"
              value={form.phone}
              onChange={handleChange}
              required
            />
            <Input
              label="Mot de passe"
              type="password"
              name="password"
              value={form.password}
              onChange={handleChange}
              required
            />
            <Input
              label="Confirmer le mot de passe"
              type="password"
              name="confirmPassword"
              value={form.confirmPassword}
              onChange={handleChange}
              required
            />
            <Button type="submit" loading={loading} className="w-full">
              S'inscrire
            </Button>
          </form>
          <p className="mt-6 text-center text-sm text-gray-500">
            Déjà un compte ?{' '}
            <Link to="/login" className="text-emerald-600 hover:text-emerald-700 font-medium">
              Se connecter
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
