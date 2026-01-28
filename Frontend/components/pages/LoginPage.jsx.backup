'use client';

import React, { useState } from 'react';
import '../styles/LoginPage.css';

export default function LoginPage({ onNavigate, updateUserData, isDarkMode, toggleTheme }) {
  const [step, setStep] = useState(1);
  const [formData, setFormData] = useState({
    server: 'federate.tech',
    username: '',
    displayName: '',
    challengeNonce: null
  });
  const [signingProgress, setSigningProgress] = useState(false);

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setFormData({ ...formData, [name]: value });
  };

  const requestChallenge = () => {
    if (formData.username) {
      const nonce = Math.random().toString(36).substring(2, 15);
      setFormData({ ...formData, challengeNonce: nonce });
      setStep(2);
    }
  };

  const simulateSigningChallenge = () => {
    setSigningProgress(true);
    setTimeout(() => {
      setSigningProgress(false);
      setStep(3);
    }, 2000);
  };

  const completeLogin = () => {
    const avatarMap = {
      'alice': '👩', 'bob': '👨', 'charlie': '👨‍💻', 'diana': '🎨', 'eve': '🔒'
    };
    
    updateUserData({
      username: formData.username,
      displayName: formData.displayName || formData.username,
      server: formData.server,
      avatar: avatarMap[formData.username] || '👤',
      email: `${formData.username}@${formData.server}`,
      bio: `Welcome to ${formData.server}!`
    });
    
    onNavigate('home');
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
          <h1 className="login-title">🔑 Cryptographic Login</h1>
          <p className="login-subtitle">Sign in with your decentralized identity</p>
        </div>

        {/* Step 1: Server & Username */}
        {step === 1 && (
          <div className="login-form">
            <div className="login-info-banner">
              <span>ℹ️</span>
              <p>Authentication uses cryptographic signing, not passwords. Your private key never leaves your device.</p>
            </div>

            <div className="form-group">
              <label htmlFor="server">Server:</label>
              <select 
                id="server"
                value={formData.server}
                onChange={handleInputChange}
                name="server"
                className="form-select"
              >
                <option>federate.tech</option>
                <option>community.social</option>
                <option>creator.hub</option>
                <option>tech-minds.io</option>
                <option>privacy-first.org</option>
              </select>
            </div>

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
            </div>

            <div className="form-group">
              <label htmlFor="displayName">Display Name (Optional):</label>
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

            <button 
              className={`btn-primary full-width ${!formData.username ? 'disabled' : ''}`}
              onClick={requestChallenge}
              disabled={!formData.username}
            >
              Request Challenge
            </button>

            <div className="login-divider">
              <span>or</span>
            </div>

            <button 
              className="btn-secondary full-width"
              onClick={() => onNavigate('signup')}
            >
              Create New Account
            </button>
          </div>
        )}

        {/* Step 2: Challenge Received */}
        {step === 2 && (
          <div className="login-form">
            <div className="challenge-box">
              <div className="challenge-header">
                <h3>🔐 Authentication Challenge</h3>
                <p>Server sent a cryptographic challenge to sign with your private key</p>
              </div>

              <div className="challenge-content">
                <div className="challenge-item">
                  <label>Server:</label>
                  <span className="challenge-value">{formData.server}</span>
                </div>

                <div className="challenge-item">
                  <label>Username:</label>
                  <span className="challenge-value">@{formData.username}</span>
                </div>

                <div className="challenge-item">
                  <label>Challenge Nonce:</label>
                  <div className="nonce-display">
                    <code>{formData.challengeNonce}</code>
                    <button 
                      className="copy-nonce-btn"
                      onClick={() => navigator.clipboard.writeText(formData.challengeNonce)}
                    >
                      📋
                    </button>
                  </div>
                </div>

                <div className="challenge-item">
                  <label>Signature Required:</label>
                  <p className="signature-note">Sign the challenge with your private key stored on this device</p>
                </div>
              </div>

              <button 
                className="btn-sign"
                onClick={simulateSigningChallenge}
              >
                🔐 Sign Challenge with Private Key
              </button>

              <button 
                className="btn-secondary full-width"
                onClick={() => setStep(1)}
              >
                ← Back
              </button>
            </div>
          </div>
        )}

        {/* Step 3: Signing Process */}
        {step === 3 && (
          <div className="login-form">
            {signingProgress ? (
              <div className="signing-progress">
                <div className="signing-spinner">
                  <div className="spinner"></div>
                </div>
                <h3>Signing Challenge...</h3>
                <p>Using WebCrypto to sign the challenge with your private key</p>
                <div className="signing-steps">
                  <div className="signing-step active">
                    <span>✓</span>
                    <span>Private key loaded from device</span>
                  </div>
                  <div className="signing-step active">
                    <span>✓</span>
                    <span>Challenge message prepared</span>
                  </div>
                  <div className="signing-step">
                    <span>→</span>
                    <span>Generating cryptographic signature...</span>
                  </div>
                </div>
              </div>
            ) : (
              <div className="signing-complete">
                <div className="success-icon">✓</div>
                <h3>Signature Generated</h3>
                <p>Your challenge has been signed successfully with your private key</p>

                <div className="signature-box">
                  <label>Signature (Sample):</label>
                  <code>
                    5Jbq7d5Hs9KmZx3Vb4JkL6XwQ8EyR7TuAoP2MnS4CdF9GhI0JkL...
                  </code>
                </div>

                <div className="security-note">
                  <span>🔒</span>
                  <p>This signature proves you control the private key for your DID. Server validates the signature against your public DID.</p>
                </div>

                <button 
                  className="btn-primary full-width"
                  onClick={completeLogin}
                >
                  Complete Login
                </button>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Flow Diagram */}
      <div className="login-flow">
        <div className="flow-step">
          <div className="flow-number">1</div>
          <div className="flow-text">
            <h4>Enter Credentials</h4>
            <p>Username & Server</p>
          </div>
        </div>

        <div className="flow-arrow">→</div>

        <div className="flow-step">
          <div className="flow-number">2</div>
          <div className="flow-text">
            <h4>Receive Challenge</h4>
            <p>Server sends nonce</p>
          </div>
        </div>

        <div className="flow-arrow">→</div>

        <div className="flow-step">
          <div className="flow-number">3</div>
          <div className="flow-text">
            <h4>Sign Locally</h4>
            <p>Private key never leaves device</p>
          </div>
        </div>

        <div className="flow-arrow">→</div>

        <div className="flow-step">
          <div className="flow-number">4</div>
          <div className="flow-text">
            <h4>Verify & Login</h4>
            <p>Server validates signature</p>
          </div>
        </div>
      </div>
    </div>
  );
}
