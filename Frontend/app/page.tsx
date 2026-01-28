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

export default function App() {
  const [currentPage, setCurrentPage] = useState('landing');
  const [isDarkMode, setIsDarkMode] = useState(true);
  const [userData, setUserData] = useState({
    username: 'alice',
    displayName: 'Alice Chen',
    bio: 'Decentralization enthusiast',
    avatar: '👩',
    email: 'alice@federate.tech',
    server: 'federate.tech',
    followers: 1250,
    following: 340,
    postsCount: 156
  });
  const [viewingUserId, setViewingUserId] = useState(null);

  useEffect(() => {
    // Apply theme to document
    if (isDarkMode) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, [isDarkMode]);

  const navigateTo = (page: string, userDataParam?: any) => {
    setCurrentPage(page);
    if (userDataParam) {
      setViewingUserId(userDataParam);
    }
  };

  const updateUserData = (newData: any) => {
    setUserData(prev => ({ ...prev, ...newData }));
  };

  const toggleTheme = () => {
    setIsDarkMode(!isDarkMode);
  };

  const sharedProps = {
    onNavigate: navigateTo,
    userData,
    updateUserData,
    isDarkMode,
    toggleTheme
  };

  return (
    <div className={`min-h-screen bg-background text-foreground ${isDarkMode ? 'dark' : ''}`}>
      {currentPage === 'landing' && <LandingPage {...sharedProps} />}
      {currentPage === 'instances' && <InstancePage {...sharedProps} />}
      {currentPage === 'signup' && <SignupPage {...sharedProps} />}
      {currentPage === 'login' && <LoginPage {...sharedProps} />}
      {currentPage === 'home' && <HomePage {...sharedProps} />}
      {currentPage === 'profile' && <ProfilePage {...sharedProps} viewingUserId={viewingUserId} />}
      {currentPage === 'thread' && <ThreadPage {...sharedProps} />}
      {currentPage === 'dm' && <DMPage {...sharedProps} />}
      {currentPage === 'security' && <SecurityPage {...sharedProps} />}
      {currentPage === 'moderation' && <ModerationPage {...sharedProps} />}
      {currentPage === 'federation' && <FederationPage {...sharedProps} />}
    </div>
  );
}
