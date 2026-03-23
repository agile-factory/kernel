import { useState } from 'react';
import { useAuthStore } from '../store/authStore';
import Input from '../components/ui/Input';
import Button from '../components/ui/Button';
import Card from '../components/ui/Card';

export default function Settings() {
  const user = useAuthStore((s) => s.user);

  const [profile, setProfile] = useState({
    first_name: user?.first_name || '',
    last_name: user?.last_name || '',
    email: user?.email || '',
    phone: user?.phone || '',
  });

  const [passwords, setPasswords] = useState({
    current_password: '',
    new_password: '',
    confirm_password: '',
  });

  const handleProfileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setProfile({ ...profile, [e.target.name]: e.target.value });
  };

  const handlePasswordChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setPasswords({ ...passwords, [e.target.name]: e.target.value });
  };

  const handleProfileSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: API integration
    alert('Profil mis à jour');
  };

  const handlePasswordSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (passwords.new_password !== passwords.confirm_password) {
      alert('Les mots de passe ne correspondent pas');
      return;
    }
    // TODO: API integration
    alert('Mot de passe mis à jour');
  };

  return (
    <div className="space-y-6 max-w-2xl">
      <h1 className="text-2xl font-bold text-gray-900">Paramètres</h1>

      <Card title="Profil">
        <form onSubmit={handleProfileSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <Input
              label="Prénom"
              name="first_name"
              value={profile.first_name}
              onChange={handleProfileChange}
              required
            />
            <Input
              label="Nom"
              name="last_name"
              value={profile.last_name}
              onChange={handleProfileChange}
              required
            />
          </div>
          <Input
            label="Email"
            type="email"
            name="email"
            value={profile.email}
            onChange={handleProfileChange}
            required
          />
          <Input
            label="Téléphone"
            name="phone"
            value={profile.phone}
            onChange={handleProfileChange}
          />
          <div className="flex justify-end">
            <Button type="submit">Enregistrer</Button>
          </div>
        </form>
      </Card>

      <Card title="Changer le mot de passe">
        <form onSubmit={handlePasswordSubmit} className="space-y-4">
          <Input
            label="Mot de passe actuel"
            type="password"
            name="current_password"
            value={passwords.current_password}
            onChange={handlePasswordChange}
            required
          />
          <Input
            label="Nouveau mot de passe"
            type="password"
            name="new_password"
            value={passwords.new_password}
            onChange={handlePasswordChange}
            required
          />
          <Input
            label="Confirmer le nouveau mot de passe"
            type="password"
            name="confirm_password"
            value={passwords.confirm_password}
            onChange={handlePasswordChange}
            required
          />
          <div className="flex justify-end">
            <Button type="submit">Enregistrer</Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
