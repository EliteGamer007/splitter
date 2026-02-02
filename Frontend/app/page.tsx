'use client';

import { useState, useEffect } from 'react';
import LandingPage from '@/components/pages/LandingPage';
import InstancePage from '@/components/pages/InstancePage';
import SignupPage from '@/components/pages/SignupPage';
import LoginPage from '@/components/pages/LoginPage';
import HomePage from '@/components/pages/HomePage';
import ProfilePage from '@/components/pages/ProfilePage';
import ThreadPage from '@/components/pages/ThreadPage';
import DMPage from '@/components/pages/DMPage';
import SecurityPage from '@/components/pages/SecurityPage';
import ModerationPage from '@/components/pages/ModerationPage';
import FederationPage from '@/components/pages/FederationPage';
import AdminPage from '@/components/pages/AdminPage';
import { userApi, healthApi } from '@/lib/api';

export default function App() {
  const [currentPage, setCurrentPage] = useState('landing');
  // Theme is handled by ThemeProvider
  const [isLoading, setIsLoading] = useState(true);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [backendConnected, setBackendConnected] = useState(false);
  const [userData, setUserData] = useState({
    id: '',
    username: '',
    displayName: '',
    bio: '',
    avatar: '👤',
    email: '',
    server: 'localhost',
    did: '',
    role: 'user',
    moderation_requested: false,
    followers: 0,
    following: 0,
    postsCount: 0
  });
  const [viewingUserId, setViewingUserId] = useState(null);
  const [selectedDMUser, setSelectedDMUser] = useState(null);

  // Check backend connection and auth state on mount
  useEffect(() => {
    const initializeApp = async () => {
      try {
        // Check backend health
        const health = await healthApi.check();
        setBackendConnected(health.status === 'ok');

        // Check if user has valid token
        const token = localStorage.getItem('jwt_token');
        if (token) {
          try {
            const user = await userApi.getCurrentUser();
            setUserData({
              id: user.id || '',
              username: user.username || '',
              displayName: user.display_name || user.username || '',
              bio: user.bio || '',
              avatar: user.avatar_url || '👤',
              email: `${user.username}@${user.instance_domain || 'localhost'}`,
              server: user.instance_domain || 'localhost',
              did: user.did || '',
              role: user.role || 'user',
              moderation_requested: user.moderation_requested || false,
              followers: user.followers_count || 0,
              following: user.following_count || 0,
              postsCount: user.posts_count || 0
            });
            setIsAuthenticated(true);
            // Redirect admin users to admin dashboard, regular users to home
            setCurrentPage(user.role === 'admin' ? 'admin' : 'home');
          } catch (authError) {
            // Token invalid, clear it
            localStorage.removeItem('jwt_token');
            setIsAuthenticated(false);
          }
        }
      } catch (error) {
        console.warn('Backend not connected:', error);
        setBackendConnected(false);
      } finally {
        setIsLoading(false);
      }
    };

    initializeApp();
  }, []);

  const navigateTo = (page: string, params?: any) => {
    // Redirect to login if trying to access protected pages without auth
    const protectedPages = ['home', 'profile', 'dm', 'security', 'moderation', 'federation', 'thread', 'admin'];
    if (protectedPages.includes(page) && !isAuthenticated) {
      setCurrentPage('login');
      return;
    }

    // Handle DM navigation with selected user
    if (page === 'dm' && params?.selectedUser) {
      setSelectedDMUser(params.selectedUser);
      setViewingUserId(null);
    } else if (page === 'profile' && params?.userId) {
      // Handle profile navigation with userId
      setViewingUserId(params.userId);
      setSelectedDMUser(null);
    } else {
      setSelectedDMUser(null);
      setViewingUserId(null);
    }

    setCurrentPage(page);
  };

  const updateUserData = (newData: any) => {
    setUserData(prev => ({ ...prev, ...newData }));
    if (newData.token) {
      setIsAuthenticated(true);
    }
  };

  const handleLogout = () => {
    localStorage.removeItem('jwt_token');
    localStorage.removeItem('private_key');
    localStorage.removeItem('public_key');
    localStorage.removeItem('did');
    setIsAuthenticated(false);
    setUserData({
      id: '',
      username: '',
      displayName: '',
      bio: '',
      avatar: '👤',
      email: '',
      server: 'localhost',
      did: '',
      role: 'user',
      moderation_requested: false,
      followers: 0,
      following: 0,
      postsCount: 0
    });
    setSelectedDMUser(null);
    setCurrentPage('landing');
  };

  const sharedProps = {
    onNavigate: navigateTo,
    userData,
    updateUserData,
    isAuthenticated,
    setIsAuthenticated,
    handleLogout,
    backendConnected
  };

  if (isLoading) {
    return (
      <div className={`min-h-screen flex items-center justify-center bg-background text-foreground`}>
        <div className="text-center">
          <div className="text-4xl mb-4">🌐</div>
          <p>Loading Splitter...</p>
        </div>
      </div>
    );
  }

  return (
    <div className={`min-h-screen bg-background text-foreground`}>
      {!backendConnected && currentPage !== 'landing' && (
        <div style={{
          background: '#ff4444',
          color: 'white',
          padding: '8px 16px',
          textAlign: 'center',
          fontSize: '14px'
        }}>
          ⚠️ Backend not connected. Please ensure the server is running at localhost:8000
        </div>
      )}
      {currentPage === 'landing' && <LandingPage {...sharedProps} />}
      {currentPage === 'instances' && <InstancePage {...sharedProps} />}
      {currentPage === 'signup' && <SignupPage {...sharedProps} />}
      {currentPage === 'login' && <LoginPage {...sharedProps} />}
      {currentPage === 'home' && <HomePage {...sharedProps} />}
      {currentPage === 'profile' && <ProfilePage {...sharedProps} viewingUserId={viewingUserId} />}
      {currentPage === 'thread' && <ThreadPage {...sharedProps} />}
      {currentPage === 'dm' && <DMPage {...sharedProps} userData={userData} selectedUser={selectedDMUser} />}
      {currentPage === 'security' && <SecurityPage {...sharedProps} />}
      {currentPage === 'moderation' && <ModerationPage {...sharedProps} />}
      {currentPage === 'federation' && <FederationPage {...sharedProps} />}
      {currentPage === 'admin' && <AdminPage {...sharedProps} />}
    </div>
  );
}
