'use client';

import React, { useState, useEffect } from 'react';
import { useTheme } from '@/components/ui/theme-provider';
import '../styles/LoginPage.css';
import { authApi, userApi } from '@/lib/api';
import { getStoredKeyPair, signChallenge, importRecoveryFile } from '@/lib/crypto';

export default function LoginPage({ onNavigate, updateUserData, setIsAuthenticated }) {
  const { theme, toggleTheme } = useTheme();
  const isDarkMode = theme === 'dark';
  const [loginMethod, setLoginMethod] = useState('password'); // 'password' or 'did'
  const [formData, setFormData] = useState({
    server: 'localhost',
    username: '',
    password: '',
    did: ''
  });
  const [keyPair, setKeyPair] = useState(null);
  const [challenge, setChallenge] = useState(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(false);
  const [step, setStep] = useState(1); // For DID login flow

  // Check for stored keys on mount
  useEffect(() => {
    const loadStoredKeys = async () => {
      try {
        const stored = await getStoredKeyPair();
        if (stored) {
          setKeyPair(stored);
          setFormData(prev => ({ ...prev, did: stored.did }));
        }
      } catch (err) {
        console.log('No stored keys found');
      }
    };
    loadStoredKeys();
  }, []);

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setFormData({ ...formData, [name]: value });
  };

  // Password Login
  const handlePasswordLogin = async (e) => {
    e.preventDefault();

    if (!formData.username || !formData.password) {
      setError('Please enter username and password');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const result = await authApi.login({
        username: formData.username,
        password: formData.password
      });

      console.log('Login successful:', result);

      // Get full user data
      const user = await userApi.getCurrentUser();
      console.log('Current user data:', user);
      console.log('User role:', user.role);

      updateUserData({
        id: user.id,
        username: user.username,
        displayName: user.display_name || user.username,
        server: user.instance_domain || formData.server,
        avatar: user.avatar_url || '👤',
        email: user.email || `${user.username}@localhost`,
        bio: user.bio || '',
        did: user.did,
        role: user.role || 'user',
        moderation_requested: user.moderation_requested || false,
        followers: user.followers_count || 0,
        following: user.following_count || 0,
        postsCount: user.posts_count || 0,
        token: result.token
      });

      if (setIsAuthenticated) {
        setIsAuthenticated(true);
      }

      setSuccess(true);
      // Use requestAnimationFrame to ensure state updates complete before navigation
      requestAnimationFrame(() => {
        onNavigate(user.role === 'admin' ? 'admin' : 'home');
      });

    } catch (err) {
      console.error('Login error:', err);
      setError(err.message || 'Login failed. Please check your credentials.');
    } finally {
      setIsLoading(false);
    }
  };

  // DID Challenge Request
  const requestChallenge = async () => {
    if (!formData.did) {
      setError('Please enter your DID');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const result = await authApi.getChallenge(formData.did);
      setChallenge(result.challenge);
      setStep(2);
    } catch (err) {
      setError(err.message || 'Failed to get challenge. Check your DID.');
    } finally {
      setIsLoading(false);
    }
  };

  // Sign challenge and verify
  const signAndVerify = async () => {
    if (!keyPair || !challenge) {
      setError('Missing key pair or challenge');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const signature = await signChallenge(keyPair.privateKey, challenge);

      const result = await authApi.verifyChallenge({
        did: formData.did,
        challenge: challenge,
        signature: signature
      });

      console.log('DID authentication successful:', result);

      // Get full user data
      const user = await userApi.getCurrentUser();

      updateUserData({
        id: user.id,
        username: user.username,
        displayName: user.display_name || user.username,
        server: user.instance_domain || formData.server,
        avatar: user.avatar_url || '👤',
        email: user.email || `${user.username}@localhost`,
        bio: user.bio || '',
        did: user.did,
        role: user.role || 'user',
        moderation_requested: user.moderation_requested || false,
        followers: user.followers_count || 0,
        following: user.following_count || 0,
        postsCount: user.posts_count || 0,
        token: result.token
      });

      if (setIsAuthenticated) {
        setIsAuthenticated(true);
      }

      setStep(3);
      // Use requestAnimationFrame to ensure state updates complete before navigation
      requestAnimationFrame(() => {
        onNavigate(user.role === 'admin' ? 'admin' : 'home');
      });

    } catch (err) {
      setError(err.message || 'Signature verification failed');
    } finally {
      setIsLoading(false);
    }
  };

  // Handle recovery file import
  const handleRecoveryImport = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    try {
      const imported = await importRecoveryFile(file);
      setKeyPair(imported);
      setFormData(prev => ({ ...prev, did: imported.did }));
      setError(null);
    } catch (err) {
      setError('Failed to import recovery file');
    }
  };

  return (
    <div className="login-container">
      <div className="login-box">
        <div className="login-header">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem' }}>
            <button
              className="login-back-btn"
              onClick={() => onNavigate('landing')}
            >
              ← Back
            </button>
            <button
              onClick={toggleTheme}
              title={isDarkMode ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
              style={{
                padding: '8px 12px',
                background: isDarkMode ? 'rgba(0, 217, 255, 0.1)' : 'rgba(100, 100, 100, 0.1)',
                border: `1px solid ${isDarkMode ? '#00d9ff' : '#666'}`,
                color: isDarkMode ? '#00d9ff' : '#333',
                borderRadius: '6px',
                cursor: 'pointer',
                fontWeight: '600'
              }}
            >
              {isDarkMode ? '🌙' : '☀️'}
            </button>
          </div>
          <h1 className="login-title">Welcome Back</h1>
          <p className="login-subtitle">Sign in to your account</p>
        </div>

        {/* Server Selection */}
        {!success && step === 1 && (
          <div className="form-group" style={{ marginBottom: '16px' }}>
            <label htmlFor="server">Server:</label>
            <select
              id="server"
              value={formData.server}
              onChange={handleInputChange}
              name="server"
              className="form-select"
            >
              <option value="localhost">localhost (Development)</option>
              <option value="federate.tech">federate.tech</option>
              <option value="community.social">community.social</option>
              <option value="creator.hub">creator.hub</option>
            </select>
          </div>
        )}

        {/* Login Method Toggle */}
        {!success && step === 1 && (
          <div style={{
            display: 'flex',
            background: 'rgba(255, 255, 255, 0.05)',
            borderRadius: '8px',
            padding: '4px',
            marginBottom: '20px'
          }}>
            <button
              onClick={() => setLoginMethod('password')}
              style={{
                flex: 1,
                padding: '10px',
                border: 'none',
                borderRadius: '6px',
                cursor: 'pointer',
                fontWeight: '600',
                background: loginMethod === 'password' ? 'linear-gradient(135deg, #00d9ff, #00ff88)' : 'transparent',
                color: loginMethod === 'password' ? '#000' : '#888'
              }}
            >
              🔑 Password
            </button>
            <button
              onClick={() => setLoginMethod('did')}
              style={{
                flex: 1,
                padding: '10px',
                border: 'none',
                borderRadius: '6px',
                cursor: 'pointer',
                fontWeight: '600',
                background: loginMethod === 'did' ? 'linear-gradient(135deg, #00d9ff, #00ff88)' : 'transparent',
                color: loginMethod === 'did' ? '#000' : '#888'
              }}
            >
              🔐 DID Challenge
            </button>
          </div>
        )}

        {error && (
          <div style={{
            background: 'rgba(255, 68, 68, 0.1)',
            border: '1px solid #ff4444',
            color: '#ff4444',
            padding: '12px',
            borderRadius: '8px',
            marginBottom: '16px'
          }}>
            ⚠️ {error}
          </div>
        )}

        {/* Password Login Form */}
        {!success && loginMethod === 'password' && step === 1 && (
          <form className="login-form" onSubmit={handlePasswordLogin}>
            <div className="form-group">
              <label htmlFor="username">Username:</label>
              <input
                id="username"
                type="text"
                name="username"
                value={formData.username}
                onChange={handleInputChange}
                placeholder="your_username"
                className="form-input"
                autoComplete="username"
                autoFocus
              />
            </div>

            <div className="form-group">
              <label htmlFor="password">Password:</label>
              <input
                id="password"
                type="password"
                name="password"
                value={formData.password}
                onChange={handleInputChange}
                placeholder="••••••••"
                className="form-input"
                autoComplete="current-password"
              />
            </div>

            <button
              type="submit"
              className={`btn-primary full-width ${isLoading ? 'disabled' : ''}`}
              disabled={isLoading}
            >
              {isLoading ? 'Signing In...' : 'Sign In'}
            </button>

            <div style={{ textAlign: 'center', marginTop: '12px' }}>
              <button
                type="button"
                onClick={() => setError('Password reset coming soon!')}
                style={{
                  background: 'none',
                  border: 'none',
                  color: '#888',
                  cursor: 'pointer',
                  textDecoration: 'underline',
                  fontSize: '14px'
                }}
              >
                Forgot password?
              </button>
            </div>
          </form>
        )}

        {/* DID Login Form - Step 1 */}
        {!success && loginMethod === 'did' && step === 1 && (
          <div className="login-form">
            <div className="login-info-banner" style={{
              background: 'rgba(0, 217, 255, 0.1)',
              border: '1px solid #00d9ff',
              borderRadius: '8px',
              padding: '12px',
              marginBottom: '16px'
            }}>
              <span>ℹ️</span>
              <p style={{ margin: '4px 0 0 0', fontSize: '14px' }}>
                DID authentication uses cryptographic signing. Your private key never leaves your device.
              </p>
            </div>

            {keyPair && (
              <div style={{
                background: 'rgba(0, 255, 136, 0.1)',
                border: '1px solid #00ff88',
                padding: '12px',
                borderRadius: '8px',
                marginBottom: '16px'
              }}>
                ✅ Found stored keys on this device
              </div>
            )}

            <div className="form-group">
              <label htmlFor="did">Your DID:</label>
              <input
                id="did"
                type="text"
                name="did"
                value={formData.did}
                onChange={handleInputChange}
                placeholder="did:key:z6Mk..."
                className="form-input"
              />
              <small>Enter your DID or use stored keys from this device</small>
            </div>

            {!keyPair && (
              <div className="form-group">
                <label style={{ display: 'block', marginBottom: '8px' }}>Import Recovery File:</label>
                <input
                  type="file"
                  accept=".json"
                  onChange={handleRecoveryImport}
                  style={{
                    width: '100%',
                    padding: '10px',
                    background: 'rgba(255, 255, 255, 0.05)',
                    border: '1px dashed #444',
                    borderRadius: '8px',
                    color: '#888'
                  }}
                />
              </div>
            )}

            <button
              className={`btn-primary full-width ${!formData.did || isLoading ? 'disabled' : ''}`}
              onClick={requestChallenge}
              disabled={!formData.did || isLoading}
            >
              {isLoading ? 'Requesting...' : 'Request Challenge'}
            </button>
          </div>
        )}

        {/* DID Login - Step 2: Sign Challenge */}
        {!success && loginMethod === 'did' && step === 2 && (
          <div className="login-form">
            <div className="challenge-box" style={{
              background: 'rgba(0, 217, 255, 0.05)',
              border: '1px solid #00d9ff',
              borderRadius: '8px',
              padding: '16px',
              marginBottom: '16px'
            }}>
              <h3 style={{ margin: '0 0 12px 0', color: '#00d9ff' }}>🔐 Authentication Challenge</h3>
              <p style={{ fontSize: '14px', color: '#888', marginBottom: '12px' }}>
                Server sent a cryptographic challenge to sign with your private key
              </p>

              <div style={{ marginBottom: '12px' }}>
                <label style={{ color: '#666', fontSize: '12px' }}>Challenge Nonce:</label>
                <div style={{
                  background: 'rgba(0, 0, 0, 0.3)',
                  padding: '10px',
                  borderRadius: '6px',
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  marginTop: '4px'
                }}>
                  <code style={{ fontSize: '12px', wordBreak: 'break-all' }}>{challenge}</code>
                  <button
                    onClick={() => navigator.clipboard.writeText(challenge)}
                    style={{
                      background: 'none',
                      border: '1px solid #00d9ff',
                      color: '#00d9ff',
                      padding: '4px 8px',
                      borderRadius: '4px',
                      cursor: 'pointer',
                      marginLeft: '8px'
                    }}
                  >
                    📋
                  </button>
                </div>
              </div>

              <div style={{ marginBottom: '12px' }}>
                <label style={{ color: '#666', fontSize: '12px' }}>Private Key Status:</label>
                <p style={{
                  margin: '4px 0 0 0',
                  color: keyPair ? '#00ff88' : '#ff4444'
                }}>
                  {keyPair ? '✅ Key loaded from device storage' : '❌ No key found - import recovery file'}
                </p>
              </div>
            </div>

            <button
              className={`btn-primary full-width ${!keyPair || isLoading ? 'disabled' : ''}`}
              onClick={signAndVerify}
              disabled={!keyPair || isLoading}
              style={{
                background: keyPair ? 'linear-gradient(135deg, #00d9ff, #00ff88)' : '#444'
              }}
            >
              {isLoading ? '🔐 Signing...' : '🔐 Sign Challenge with Private Key'}
            </button>

            <button
              className="btn-secondary full-width"
              onClick={() => { setStep(1); setChallenge(null); setError(null); }}
              style={{ marginTop: '12px' }}
            >
              ← Back
            </button>
          </div>
        )}

        {/* Success State */}
        {(success || step === 3) && (
          <div className="login-form">
            <div className="signing-complete" style={{ textAlign: 'center', padding: '40px 0' }}>
              <div className="success-icon" style={{
                width: '80px',
                height: '80px',
                borderRadius: '50%',
                background: 'linear-gradient(135deg, #00d9ff, #00ff88)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                margin: '0 auto 20px',
                fontSize: '40px'
              }}>✓</div>
              <h3 style={{ color: '#00d9ff', marginBottom: '8px' }}>Welcome Back!</h3>
              <p style={{ color: '#888' }}>
                {loginMethod === 'did' ? 'Identity verified via cryptographic signature' : 'Login successful'}
              </p>
              <p style={{ color: '#666', marginTop: '16px' }}>Redirecting to home...</p>
            </div>
          </div>
        )}

        {/* Create Account Link */}
        {!success && step === 1 && (
          <>
            <div className="login-divider" style={{
              display: 'flex',
              alignItems: 'center',
              margin: '20px 0',
              color: '#666'
            }}>
              <div style={{ flex: 1, height: '1px', background: '#333' }} />
              <span style={{ padding: '0 12px' }}>or</span>
              <div style={{ flex: 1, height: '1px', background: '#333' }} />
            </div>

            <button
              className="btn-secondary full-width"
              onClick={() => onNavigate('signup')}
            >
              Create New Account
            </button>
          </>
        )}
      </div>

      {/* Features showcase */}
      <div className="login-features" style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(3, 1fr)',
        gap: '20px',
        marginTop: '40px',
        maxWidth: '600px'
      }}>
        <div className="feature-item" style={{ textAlign: 'center' }}>
          <div style={{ fontSize: '32px', marginBottom: '8px' }}>🔒</div>
          <h4 style={{ color: '#00d9ff', margin: '0 0 4px 0', fontSize: '14px' }}>Secure</h4>
          <p style={{ color: '#666', fontSize: '12px', margin: 0 }}>Your data is encrypted</p>
        </div>
        <div className="feature-item" style={{ textAlign: 'center' }}>
          <div style={{ fontSize: '32px', marginBottom: '8px' }}>🌐</div>
          <h4 style={{ color: '#00d9ff', margin: '0 0 4px 0', fontSize: '14px' }}>Federated</h4>
          <p style={{ color: '#666', fontSize: '12px', margin: 0 }}>Connect anywhere</p>
        </div>
        <div className="feature-item" style={{ textAlign: 'center' }}>
          <h4 style={{ color: '#00d9ff', margin: '0 0 4px 0', fontSize: '14px' }}>Open</h4>
          <p style={{ color: '#666', fontSize: '12px', margin: 0 }}>You own your identity</p>
        </div>
      </div>
    </div>
  );
}
