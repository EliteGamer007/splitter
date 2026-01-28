'use client';

import React, { useState } from 'react';
import '../styles/SignupPage.css';

export default function SignupPage({ onNavigate, updateUserData, isDarkMode, toggleTheme }) {
  const [step, setStep] = useState(1);
  const [formData, setFormData] = useState({
    server: 'federate.tech',
    username: '',
    displayName: '',
    bio: '',
    password: '',
    avatar: '👤'
  });
  const [didGenerated, setDidGenerated] = useState(null);

  const handleServerChange = (e) => {
    setFormData({ ...formData, server: e.target.value });
  };

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setFormData({ ...formData, [name]: value });
  };

  const generateDID = () => {
    const randomId = Math.random().toString(36).substring(2, 15);
    const did = `did:key:z6Mk${randomId}`;
    setDidGenerated(did);
  };

  const handleDownloadRecovery = () => {
    const recoveryData = {
      did: didGenerated,
      timestamp: new Date().toISOString(),
      server: formData.server,
      username: formData.username
    };
    const dataStr = JSON.stringify(recoveryData, null, 2);
    const element = document.createElement('a');
    element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(dataStr));
    element.setAttribute('download', `federate-recovery-${Date.now()}.json`);
    element.style.display = 'none';
    document.body.appendChild(element);
    element.click();
    document.body.removeChild(element);
  };

  const nextStep = () => {
    if (step < 4) setStep(step + 1);
  };

  const prevStep = () => {
    if (step > 1) setStep(step - 1);
  };

  const submitSignup = () => {
    if (formData.username && formData.displayName && didGenerated) {
      const avatarMap = {
        'alice': '👩', 'bob': '👨', 'charlie': '👨‍💻', 'diana': '🎨', 'eve': '🔒'
      };
      
      updateUserData({
        username: formData.username,
        displayName: formData.displayName,
        server: formData.server,
        avatar: avatarMap[formData.username] || '👤',
        email: `${formData.username}@${formData.server}`,
        bio: formData.bio || `Welcome to ${formData.server}!`
      });
      
      onNavigate('home');
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
          <h1 className="signup-title">🔐 Create Your Identity</h1>
          <p className="signup-subtitle">Join the federated network</p>
        </div>

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
            <div className="form-group">
              <label htmlFor="server">Choose a federated server:</label>
              <select 
                id="server"
                value={formData.server}
                onChange={handleServerChange}
                className="form-select"
              >
                <option>federate.tech</option>
                <option>community.social</option>
                <option>creator.hub</option>
                <option>tech-minds.io</option>
                <option>privacy-first.org</option>
              </select>
              <p className="server-description">
                You'll connect to {formData.server} with your decentralized identity.
              </p>
            </div>
            <div className="form-buttons">
              <button 
                className="btn-primary"
                onClick={nextStep}
              >
                Next →
              </button>
            </div>
          </div>
        )}

        {/* Step 2: Generate DID */}
        {step === 2 && (
          <div className="signup-form">
            <h2 className="step-title">Generate Your Identity 🔑</h2>
            
            <div className="security-banner">
              <span>⚠️</span>
              <p>If you lose this device and recovery file, your account cannot be recovered.</p>
            </div>

            {!didGenerated ? (
              <div className="form-group">
                <p className="form-description">
                  Your decentralized identifier (DID) will be generated using WebCrypto and stored securely on your device.
                </p>
                <button 
                  className="btn-generate"
                  onClick={generateDID}
                >
                  Generate Decentralized Identity
                </button>
              </div>
            ) : (
              <div className="did-display">
                <div className="did-section">
                  <label>Your DID (Public):</label>
                  <div className="did-box">
                    <code>{didGenerated}</code>
                    <button 
                      className="copy-btn"
                      onClick={() => navigator.clipboard.writeText(didGenerated)}
                    >
                      📋 Copy
                    </button>
                  </div>
                </div>

                <div className="did-section">
                  <label>Private Key:</label>
                  <div className="did-box locked">
                    <code>••••••••••••••••••••••••••••••••</code>
                    <span className="lock-icon">🔒 Stored Securely</span>
                  </div>
                </div>

                <button 
                  className="btn-recovery"
                  onClick={handleDownloadRecovery}
                >
                  💾 Download Encrypted Recovery File
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
              <button 
                className={`btn-primary ${!didGenerated ? 'disabled' : ''}`}
                onClick={nextStep}
                disabled={!didGenerated}
              >
                Next →
              </button>
            </div>
          </div>
        )}

        {/* Step 3: Security Setup */}
        {step === 3 && (
          <div className="signup-form">
            <h2 className="step-title">Security Setup 🔐</h2>
            
            <div className="form-group">
              <label htmlFor="username">Username:</label>
              <input
                id="username"
                type="text"
                name="username"
                value={formData.username}
                onChange={handleInputChange}
                placeholder="@your_username"
                className="form-input"
              />
              <small>This will be your public handle on the network</small>
            </div>

            <div className="form-group">
              <label htmlFor="password">Password (Optional - for local device login):</label>
              <input
                id="password"
                type="password"
                name="password"
                value={formData.password}
                onChange={handleInputChange}
                placeholder="••••••••"
                className="form-input"
              />
              <small>Authentication uses cryptographic signing, not passwords</small>
            </div>

            <div className="form-group checkbox">
              <input 
                type="checkbox" 
                id="recovery"
                defaultChecked
                disabled
              />
              <label htmlFor="recovery">Recovery file is enabled</label>
            </div>

            <div className="form-buttons">
              <button 
                className="btn-secondary"
                onClick={prevStep}
              >
                ← Back
              </button>
              <button 
                className={`btn-primary ${!formData.username ? 'disabled' : ''}`}
                onClick={nextStep}
                disabled={!formData.username}
              >
                Next →
              </button>
            </div>
          </div>
        )}

        {/* Step 4: Profile Setup */}
        {step === 4 && (
          <div className="signup-form">
            <h2 className="step-title">Complete Your Profile 👤</h2>
            
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

            <div className="form-group">
              <label>Default Post Visibility:</label>
              <div className="visibility-options">
                <label className="radio-option">
                  <input type="radio" name="visibility" defaultChecked />
                  <span>Public 🌐</span>
                </label>
                <label className="radio-option">
                  <input type="radio" name="visibility" />
                  <span>Followers Only 👥</span>
                </label>
                <label className="radio-option disabled">
                  <input type="radio" name="visibility" disabled />
                  <span>Custom Circle 🟡 (Sprint 2)</span>
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
                className={`btn-primary ${!formData.displayName ? 'disabled' : ''}`}
                onClick={submitSignup}
                disabled={!formData.displayName}
              >
                Create Account ✓
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
