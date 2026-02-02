'use client';

import React, { useState } from 'react';
import { useTheme } from '@/components/ui/theme-provider';
import '../styles/SignupPage.css';
import { authApi } from '@/lib/api';
import { generateKeyPair, storeKeyPair, exportRecoveryFile } from '@/lib/crypto';

// Server data with locations
const SERVERS = [
  {
    id: 1,
    name: 'delhi-hub.in',
    category: 'General Community',
    users: 4350,
    federation: 'Open',
    moderation: 'Moderate',
    reputation: 'trusted',
    description: 'Community hub for Delhi region - culture, tech, and local news',
    region: 'Delhi',
    location: 'Delhi, India',
    uptime: '99.9%',
    ping: '12ms'
  },
  {
    id: 2,
    name: 'bangalore-tech.in',
    category: 'Technology',
    users: 6200,
    federation: 'Open',
    moderation: 'Strict',
    reputation: 'trusted',
    description: 'Tech hub for Bangalore - startups, developers, and innovation',
    region: 'Karnataka',
    location: 'Bangalore, India',
    uptime: '99.8%',
    ping: '15ms'
  },
  {
    id: 3,
    name: 'mumbai-creative.in',
    category: 'Creative & Arts',
    users: 5100,
    federation: 'Open',
    moderation: 'Lenient',
    reputation: 'trusted',
    description: 'Creative hub for Mumbai - artists, filmmakers, and creators',
    region: 'Maharashtra',
    location: 'Mumbai, India',
    uptime: '99.7%',
    ping: '18ms'
  },
  {
    id: 4,
    name: 'kolkata-academic.in',
    category: 'Academic & Research',
    users: 2800,
    federation: 'Open',
    moderation: 'Strict',
    reputation: 'trusted',
    description: 'Academic community for Eastern India - research and education',
    region: 'West Bengal',
    location: 'Kolkata, India',
    uptime: '99.5%',
    ping: '22ms'
  },
  {
    id: 5,
    name: 'hyderabad-network.in',
    category: 'Technology',
    users: 3900,
    federation: 'Open',
    moderation: 'Moderate',
    reputation: 'trusted',
    description: 'Tech community for Hyderabad - AI, ML, and software development',
    region: 'Telangana',
    location: 'Hyderabad, India',
    uptime: '99.6%',
    ping: '16ms'
  },
  {
    id: 6,
    name: 'localhost',
    category: 'Development',
    users: 10,
    federation: 'Open',
    moderation: 'None',
    reputation: 'dev',
    description: 'Local development server for testing',
    region: 'Local',
    location: 'Local Machine',
    uptime: '100%',
    ping: '1ms'
  }
];

const REGIONS = ['All', 'Delhi', 'Karnataka', 'Maharashtra', 'West Bengal', 'Telangana', 'Local'];

export default function SignupPage({ onNavigate, updateUserData, setIsAuthenticated }) {
  const { theme, toggleTheme } = useTheme();
  const isDarkMode = theme === 'dark';
  const [step, setStep] = useState(1);
  const [formData, setFormData] = useState({
    server: '',
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
    displayName: '',
    bio: '',
    avatar: '👤'
  });
  const [keyPair, setKeyPair] = useState(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(null);
  const [passwordStrength, setPasswordStrength] = useState(0);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedRegion, setSelectedRegion] = useState('All');

  // Filter servers based on search and region
  const filteredServers = SERVERS.filter(server => {
    const matchesSearch = server.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      server.category.toLowerCase().includes(searchTerm.toLowerCase()) ||
      server.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesRegion = selectedRegion === 'All' || server.region === selectedRegion;
    return matchesSearch && matchesRegion;
  });

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setFormData({ ...formData, [name]: value });

    // Check password strength
    if (name === 'password') {
      let strength = 0;
      if (value.length >= 8) strength++;
      if (/[A-Z]/.test(value)) strength++;
      if (/[a-z]/.test(value)) strength++;
      if (/[0-9]/.test(value)) strength++;
      if (/[^A-Za-z0-9]/.test(value)) strength++;
      setPasswordStrength(strength);
    }
  };

  const handleServerChange = (e) => {
    setFormData({ ...formData, server: e.target.value });
  };

  const handleGenerateDID = async () => {
    setIsLoading(true);
    setError(null);

    try {
      const newKeyPair = await generateKeyPair();
      setKeyPair(newKeyPair);
      // Store the key pair for this session
      await storeKeyPair(newKeyPair);
    } catch (err) {
      setError('Failed to generate cryptographic keys. Please try again.');
      console.error('Key generation error:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleDownloadRecovery = () => {
    if (!keyPair || !formData.username) return;

    try {
      exportRecoveryFile(keyPair, formData.username, formData.server);
    } catch (err) {
      setError('Failed to create recovery file');
    }
  };

  const getPasswordStrengthColor = () => {
    if (passwordStrength <= 1) return '#ff4444';
    if (passwordStrength <= 2) return '#ffaa00';
    if (passwordStrength <= 3) return '#ffff00';
    return '#00ff00';
  };

  const getPasswordStrengthText = () => {
    if (passwordStrength <= 1) return 'Weak';
    if (passwordStrength <= 2) return 'Fair';
    if (passwordStrength <= 3) return 'Good';
    return 'Strong';
  };

  const validateStep2 = () => {
    if (!formData.username || formData.username.length < 3) {
      setError('Username must be at least 3 characters');
      return false;
    }
    if (!/^[a-zA-Z0-9_]+$/.test(formData.username)) {
      setError('Username can only contain letters, numbers, and underscores');
      return false;
    }
    if (!formData.email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      setError('Please enter a valid email address');
      return false;
    }
    if (!formData.password || formData.password.length < 8) {
      setError('Password must be at least 8 characters');
      return false;
    }
    if (formData.password !== formData.confirmPassword) {
      setError('Passwords do not match');
      return false;
    }
    return true;
  };

  const nextStep = () => {
    setError(null);
    if (step === 2 && !validateStep2()) {
      return;
    }
    setStep(step + 1);
  };

  const prevStep = () => {
    setError(null);
    setStep(step - 1);
  };

  const submitSignup = async () => {
    if (!formData.displayName) {
      setError('Please enter a display name');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      // Register with backend - include DID if generated, otherwise backend auto-generates
      const registrationData = {
        username: formData.username,
        email: formData.email,
        password: formData.password,
        display_name: formData.displayName,
        bio: formData.bio,
        instance_domain: formData.server
      };

      // If user generated a DID, include it
      if (keyPair) {
        registrationData.did = keyPair.did;
        registrationData.public_key = keyPair.publicKeyBase64;
      }

      const result = await authApi.register(registrationData);
      console.log('Registration successful:', result);

      // Move to success step
      setStep(5);

      // Redirect to login after success
      setTimeout(() => {
        onNavigate('login');
      }, 2000);

    } catch (err) {
      console.error('Registration error:', err);
      setError(err.message || 'Registration failed. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="signup-container">
      <div className="signup-box">
        <div className="signup-header">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem' }}>
            <button
              className="signup-back-btn"
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
          <h1 className="signup-title">Create Your Identity</h1>
          <p className="signup-subtitle">Set up your decentralized account</p>
        </div>

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

        {/* Progress Bar */}
        <div className="signup-progress">
          <div className="progress-bar">
            <div
              className="progress-fill"
              style={{ width: `${(step / 4) * 100}%` }}
            />
          </div>
          <div className="progress-steps">
            <div className={`step ${step >= 1 ? 'active' : ''}`}>1</div>
            <div className={`step ${step >= 2 ? 'active' : ''}`}>2</div>
            <div className={`step ${step >= 3 ? 'active' : ''}`}>3</div>
            <div className={`step ${step >= 4 ? 'active' : ''}`}>4</div>
          </div>
        </div>

        {/* Step 1: Select Server */}
        {step === 1 && (
          <div className="signup-form">
            <h2 className="step-title">Select Your Server 🧭</h2>
            <p style={{ color: '#888', marginBottom: '16px', fontSize: '14px' }}>
              Choose a federated server aligned with your interests and region
            </p>

            {/* Search & Filter */}
            <div style={{ display: 'flex', gap: '12px', marginBottom: '20px', flexWrap: 'wrap' }}>
              <div style={{ flex: 1, minWidth: '200px', position: 'relative' }}>
                <span style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', color: '#666' }}>🔍</span>
                <input
                  type="text"
                  placeholder="Search servers..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  style={{
                    width: '100%',
                    padding: '12px 12px 12px 40px',
                    background: 'rgba(255,255,255,0.05)',
                    border: '1px solid #333',
                    borderRadius: '8px',
                    color: '#fff',
                    fontSize: '14px'
                  }}
                />
              </div>
              <select
                value={selectedRegion}
                onChange={(e) => setSelectedRegion(e.target.value)}
                style={{
                  padding: '12px 16px',
                  background: 'rgba(255,255,255,0.05)',
                  border: '1px solid #333',
                  borderRadius: '8px',
                  color: '#fff',
                  fontSize: '14px',
                  minWidth: '150px'
                }}
              >
                {REGIONS.map(region => (
                  <option key={region} value={region} style={{ background: '#1a1a2e' }}>{region}</option>
                ))}
              </select>
            </div>

            {/* Server Cards Grid */}
            <div style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
              gap: '16px',
              maxHeight: '400px',
              overflowY: 'auto',
              padding: '4px'
            }}>
              {filteredServers.map(server => (
                <div
                  key={server.id}
                  onClick={() => setFormData({ ...formData, server: server.name })}
                  style={{
                    background: formData.server === server.name
                      ? 'linear-gradient(135deg, rgba(0, 217, 255, 0.2), rgba(0, 255, 136, 0.1))'
                      : 'rgba(255,255,255,0.03)',
                    border: formData.server === server.name
                      ? '2px solid #00d9ff'
                      : '1px solid #333',
                    borderRadius: '12px',
                    padding: '16px',
                    cursor: 'pointer',
                    transition: 'all 0.2s ease',
                    position: 'relative'
                  }}
                >
                  {/* Selected indicator */}
                  {formData.server === server.name && (
                    <div style={{
                      position: 'absolute',
                      top: '12px',
                      right: '12px',
                      width: '24px',
                      height: '24px',
                      background: 'linear-gradient(135deg, #00d9ff, #00ff88)',
                      borderRadius: '50%',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: '14px'
                    }}>✓</div>
                  )}

                  {/* Server Header */}
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                    <span style={{ fontSize: '20px' }}>
                      {server.reputation === 'trusted' ? '🟢' : server.reputation === 'dev' ? '🔧' : '🟡'}
                    </span>
                    <h3 style={{ margin: 0, color: '#fff', fontSize: '16px', fontWeight: '600' }}>
                      {server.name}
                    </h3>
                  </div>

                  {/* Location */}
                  <div style={{ color: '#00d9ff', fontSize: '13px', marginBottom: '8px' }}>
                    {server.location}
                  </div>

                  {/* Category Badge */}
                  <div style={{
                    display: 'inline-block',
                    padding: '4px 8px',
                    background: 'rgba(0, 217, 255, 0.1)',
                    border: '1px solid rgba(0, 217, 255, 0.3)',
                    borderRadius: '4px',
                    fontSize: '11px',
                    color: '#00d9ff',
                    marginBottom: '8px'
                  }}>
                    {server.category}
                  </div>

                  {/* Description */}
                  <p style={{ color: '#888', fontSize: '12px', margin: '8px 0', lineHeight: '1.4' }}>
                    {server.description}
                  </p>

                  {/* Stats */}
                  <div style={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(3, 1fr)',
                    gap: '8px',
                    marginTop: '12px',
                    paddingTop: '12px',
                    borderTop: '1px solid #333'
                  }}>
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ color: '#666', fontSize: '10px', textTransform: 'uppercase' }}>Users</div>
                      <div style={{ color: '#fff', fontSize: '14px', fontWeight: '600' }}>
                        {server.users.toLocaleString()}
                      </div>
                    </div>
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ color: '#666', fontSize: '10px', textTransform: 'uppercase' }}>Uptime</div>
                      <div style={{ color: '#00ff88', fontSize: '14px', fontWeight: '600' }}>
                        {server.uptime}
                      </div>
                    </div>
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ color: '#666', fontSize: '10px', textTransform: 'uppercase' }}>Ping</div>
                      <div style={{ color: '#00d9ff', fontSize: '14px', fontWeight: '600' }}>
                        {server.ping}
                      </div>
                    </div>
                  </div>

                  {/* Moderation Badge */}
                  <div style={{
                    marginTop: '12px',
                    padding: '6px 10px',
                    background: server.moderation === 'Strict' ? 'rgba(255, 68, 68, 0.1)' :
                      server.moderation === 'Moderate' ? 'rgba(255, 170, 0, 0.1)' :
                        'rgba(0, 255, 136, 0.1)',
                    border: `1px solid ${server.moderation === 'Strict' ? 'rgba(255, 68, 68, 0.3)' :
                      server.moderation === 'Moderate' ? 'rgba(255, 170, 0, 0.3)' :
                        'rgba(0, 255, 136, 0.3)'}`,
                    borderRadius: '6px',
                    fontSize: '11px',
                    color: server.moderation === 'Strict' ? '#ff4444' :
                      server.moderation === 'Moderate' ? '#ffaa00' : '#00ff88',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '6px'
                  }}>
                    <span>{server.moderation === 'Strict' ? '🛡️' : server.moderation === 'Moderate' ? '⚖️' : '🌿'}</span>
                    {server.moderation} Moderation • {server.federation} Federation
                  </div>
                </div>
              ))}
            </div>

            {filteredServers.length === 0 && (
              <div style={{ textAlign: 'center', padding: '40px', color: '#666' }}>
                <p>No servers found matching your criteria</p>
              </div>
            )}

            <div className="form-buttons" style={{ marginTop: '20px' }}>
              <button
                className={`btn-primary ${!formData.server ? 'disabled' : ''}`}
                onClick={nextStep}
                disabled={!formData.server}
                style={{
                  background: formData.server ? 'linear-gradient(135deg, #00d9ff, #00ff88)' : '#333',
                  color: formData.server ? '#000' : '#666'
                }}
              >
                Continue with {formData.server || 'selected server'} →
              </button>
            </div>
          </div>
        )}

        {/* Step 2: Account Credentials */}
        {step === 2 && (
          <div className="signup-form">
            <h2 className="step-title">Account Credentials 🔑</h2>

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
              />
              <small>This will be your public handle (letters, numbers, underscores)</small>
            </div>

            <div className="form-group">
              <label htmlFor="email">Email:</label>
              <input
                id="email"
                type="email"
                name="email"
                value={formData.email}
                onChange={handleInputChange}
                placeholder="you@example.com"
                className="form-input"
                autoComplete="email"
              />
              <small>For account recovery and notifications</small>
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
                autoComplete="new-password"
              />
              {formData.password && (
                <div style={{ marginTop: '8px' }}>
                  <div style={{
                    height: '4px',
                    background: '#333',
                    borderRadius: '2px',
                    overflow: 'hidden'
                  }}>
                    <div style={{
                      width: `${(passwordStrength / 5) * 100}%`,
                      height: '100%',
                      background: getPasswordStrengthColor(),
                      transition: 'all 0.3s'
                    }} />
                  </div>
                  <small style={{ color: getPasswordStrengthColor() }}>
                    Password strength: {getPasswordStrengthText()}
                  </small>
                </div>
              )}
            </div>

            <div className="form-group">
              <label htmlFor="confirmPassword">Confirm Password:</label>
              <input
                id="confirmPassword"
                type="password"
                name="confirmPassword"
                value={formData.confirmPassword}
                onChange={handleInputChange}
                placeholder="••••••••"
                className="form-input"
                autoComplete="new-password"
              />
              {formData.confirmPassword && formData.password !== formData.confirmPassword && (
                <small style={{ color: '#ff4444' }}>Passwords do not match</small>
              )}
              {formData.confirmPassword && formData.password === formData.confirmPassword && formData.password && (
                <small style={{ color: '#00ff00' }}>✓ Passwords match</small>
              )}
            </div>

            <div className="form-buttons">
              <button
                className="btn-secondary"
                onClick={prevStep}
              >
                ← Back
              </button>
              <button
                className="btn-primary"
                onClick={nextStep}
              >
                Next →
              </button>
            </div>
          </div>
        )}

        {/* Step 3: Generate DID (Optional) */}
        {step === 3 && (
          <div className="signup-form">
            <h2 className="step-title">Decentralized Identity</h2>
            <div className="security-banner" style={{
              background: 'rgba(0, 217, 255, 0.1)',
              border: '1px solid #00d9ff',
              borderRadius: '8px',
              padding: '12px',
              marginBottom: '16px'
            }}>
              <span>ℹ️</span>
              <p style={{ margin: '4px 0 0 0', fontSize: '14px' }}>
                Generate a DID for enhanced security with cryptographic authentication.
                This is <strong>optional</strong> - you can still use password login.
              </p>
            </div>

            {!keyPair ? (
              <div className="form-group">
                <p className="form-description">
                  Your decentralized identifier (DID) will be generated using WebCrypto and stored securely on your device.
                </p>
                <button
                  className="btn-generate"
                  onClick={handleGenerateDID}
                  disabled={isLoading}
                  style={{
                    width: '100%',
                    padding: '14px',
                    background: 'linear-gradient(135deg, #00d9ff, #00ff88)',
                    border: 'none',
                    borderRadius: '8px',
                    color: '#000',
                    fontWeight: '600',
                    cursor: 'pointer',
                    marginBottom: '12px'
                  }}
                >
                  {isLoading ? 'Generating...' : '🔑 Generate Decentralized Identity'}
                </button>
                <button
                  className="btn-secondary"
                  onClick={nextStep}
                  style={{ width: '100%' }}
                >
                  Skip - Use Password Only →
                </button>
              </div>
            ) : (
              <div className="did-display">
                <div className="did-section" style={{ marginBottom: '16px' }}>
                  <label style={{ color: '#00d9ff', fontWeight: '600' }}>✅ Your DID (Public):</label>
                  <div className="did-box" style={{
                    background: 'rgba(0, 255, 136, 0.1)',
                    border: '1px solid #00ff88',
                    borderRadius: '8px',
                    padding: '12px',
                    marginTop: '8px',
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center'
                  }}>
                    <code style={{ fontSize: '12px', wordBreak: 'break-all' }}>{keyPair.did}</code>
                    <button
                      className="copy-btn"
                      onClick={() => navigator.clipboard.writeText(keyPair.did)}
                      style={{
                        background: 'none',
                        border: '1px solid #00ff88',
                        color: '#00ff88',
                        padding: '4px 8px',
                        borderRadius: '4px',
                        cursor: 'pointer',
                        marginLeft: '8px'
                      }}
                    >
                      📋 Copy
                    </button>
                  </div>
                </div>

                <div className="did-section" style={{ marginBottom: '16px' }}>
                  <label style={{ color: '#888' }}>Private Key:</label>
                  <div className="did-box locked" style={{
                    background: 'rgba(255, 255, 255, 0.05)',
                    border: '1px solid #444',
                    borderRadius: '8px',
                    padding: '12px',
                    marginTop: '8px',
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center'
                  }}>
                    <code>••••••••••••••••••••••••••••••••</code>
                    <span className="lock-icon">🔒 Stored Securely</span>
                  </div>
                </div>

                <div className="security-banner" style={{
                  background: 'rgba(255, 170, 0, 0.1)',
                  border: '1px solid #ffaa00',
                  borderRadius: '8px',
                  padding: '12px',
                  marginBottom: '16px'
                }}>
                  <span>⚠️</span>
                  <p style={{ margin: '4px 0 0 0', fontSize: '14px' }}>
                    Download your recovery file! If you lose this device, you'll need it to recover your DID.
                  </p>
                </div>

                <button
                  className="btn-recovery"
                  onClick={handleDownloadRecovery}
                  disabled={!formData.username}
                  style={{
                    width: '100%',
                    padding: '12px',
                    background: 'rgba(255, 170, 0, 0.2)',
                    border: '1px solid #ffaa00',
                    borderRadius: '8px',
                    color: '#ffaa00',
                    fontWeight: '600',
                    cursor: formData.username ? 'pointer' : 'not-allowed',
                    marginBottom: '16px'
                  }}
                >
                  💾 Download Recovery File
                </button>
              </div>
            )}

            <div className="form-buttons">
              <button
                className="btn-secondary"
                onClick={prevStep}
              >
                ← Back
              </button>
              {keyPair && (
                <button
                  className="btn-primary"
                  onClick={nextStep}
                >
                  Next →
                </button>
              )}
            </div>
          </div>
        )}

        {/* Step 4: Profile Setup */}
        {step === 4 && (
          <div className="signup-form">
            <h2 className="step-title">Complete Your Profile</h2>
            <div className="form-group">
              <label htmlFor="displayName">Display Name:</label>
              <input
                id="displayName"
                type="text"
                name="displayName"
                value={formData.displayName}
                onChange={handleInputChange}
                placeholder="Your Full Name"
                className="form-input"
              />
              <small>This is how others will see you</small>
            </div>

            <div className="form-group">
              <label htmlFor="bio">Bio (Optional):</label>
              <textarea
                id="bio"
                name="bio"
                value={formData.bio}
                onChange={handleInputChange}
                placeholder="Tell us about yourself..."
                className="form-textarea"
                rows="4"
              />
            </div>

            <div className="account-summary" style={{
              background: 'rgba(0, 217, 255, 0.1)',
              border: '1px solid #00d9ff',
              borderRadius: '8px',
              padding: '16px',
              marginBottom: '16px'
            }}>
              <h4 style={{ margin: '0 0 12px 0', color: '#00d9ff' }}>Account Summary</h4>
              <p style={{ margin: '4px 0', fontSize: '14px' }}>
                <strong>Server:</strong> {formData.server}
              </p>
              <p style={{ margin: '4px 0', fontSize: '14px' }}>
                <strong>Username:</strong> @{formData.username}
              </p>
              <p style={{ margin: '4px 0', fontSize: '14px' }}>
                <strong>Email:</strong> {formData.email}
              </p>
              <p style={{ margin: '4px 0', fontSize: '14px' }}>
                <strong>DID:</strong> {keyPair ? '✅ Generated' : '⏭️ Auto-generated on server'}
              </p>
              <p style={{ margin: '4px 0', fontSize: '14px' }}>
                <strong>Auth:</strong> Password {keyPair ? '+ DID Challenge' : 'only'}
              </p>
            </div>

            <div className="form-group">
              <label>Default Post Visibility:</label>
              <div className="visibility-options" style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginTop: '8px' }}>
                <label className="radio-option" style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <input type="radio" name="visibility" defaultChecked />
                  <span>Public 🌐</span>
                </label>
                <label className="radio-option" style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <input type="radio" name="visibility" />
                  <span>Followers Only 👥</span>
                </label>
                <label className="radio-option disabled" style={{ display: 'flex', alignItems: 'center', gap: '8px', opacity: 0.5 }}>
                  <input type="radio" name="visibility" disabled />
                  <span>Custom Circle 🟡 (Coming Soon)</span>
                </label>
              </div>
            </div>

            <div className="form-buttons">
              <button
                className="btn-secondary"
                onClick={prevStep}
              >
                ← Back
              </button>
              <button
                className={`btn-primary ${!formData.displayName || isLoading ? 'disabled' : ''}`}
                onClick={submitSignup}
                disabled={!formData.displayName || isLoading}
              >
                {isLoading ? 'Creating Account...' : 'Create Account ✓'}
              </button>
            </div>
          </div>
        )}

        {/* Step 5: Success */}
        {step === 5 && (
          <div className="signup-form">
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
              <h3 style={{ color: '#00d9ff', marginBottom: '8px' }}>Account Created!</h3>
              <p style={{ color: '#888' }}>Welcome to Splitter, @{formData.username}</p>
              <p style={{ color: '#666', marginTop: '8px', fontSize: '14px' }}>
                Server: {formData.server}
              </p>
              <p style={{ color: '#666', marginTop: '16px' }}>Redirecting to login...</p>
            </div>
          </div>
        )}

        {/* Link to login */}
        {step < 5 && (
          <div style={{ textAlign: 'center', marginTop: '16px' }}>
            Already have an account?{' '}
            <button
              onClick={() => onNavigate('login')}
              style={{
                background: 'none',
                border: 'none',
                color: '#00d9ff',
                cursor: 'pointer',
                textDecoration: 'underline'
              }}
            >
              Sign In
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
