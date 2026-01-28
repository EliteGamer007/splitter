'use client';

import { useState } from 'react';
import '../styles/SecurityPage.css';

export default function SecurityPage({ onNavigate, isDarkMode, toggleTheme }) {
  const [showRecoveryCode, setShowRecoveryCode] = useState(false);
  const [copiedField, setCopiedField] = useState(null);

  const recoveryCode = 'RECOVERY_CODE_9c8f7e6d5c4b3a2f1e0d9c8b7a6f5e4d';
  const did = 'did:key:z6Mkjx9J5aQ2vP7nL4mK8vQ5wR2xT9nY6bZ3cD5eF7gH9...';
  const publicKeyFingerprint = 'A4:9C:2B:7D:E1:5F:8A:3C:9B:6E:2A:4D:7C:1F:8E:5B';

  const handleCopyToClipboard = (text, fieldName) => {
    navigator.clipboard.writeText(text);
    setCopiedField(fieldName);
    setTimeout(() => setCopiedField(null), 2000);
  };

  return (
    <div className="security-container">
      {/* Navigation */}
      <div className="security-navbar">
        <button
          className="nav-button back-button"
          onClick={() => onNavigate('home')}
        >
          ← Back
        </button>
        <h1 className="navbar-title">Security Dashboard</h1>
        <div style={{ marginLeft: 'auto', display: 'flex', gap: '10px' }}>
          <button onClick={() => onNavigate('profile')} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>👤 Profile</button>
          <button onClick={() => onNavigate('dm')} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>💬 Messages</button>
          <button onClick={toggleTheme} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>{isDarkMode ? '🌙' : '☀️'}</button>
        </div>
      </div>

      <div className="security-content">
        {/* Intro Banner */}
        <div className="security-banner">
          <div className="banner-icon">🔐</div>
          <div className="banner-text">
            <h2>Client-Side Key Custody</h2>
            <p>
              This device controls your identity. Your private key is stored
              only on this device. Losing access means losing your account.
            </p>
          </div>
        </div>

        {/* Identity Status Card */}
        <div className="status-card">
          <h3 className="card-title">Identity Status</h3>
          <div className="status-items">
            <div className="status-item">
              <div className="status-check">✔</div>
              <div className="status-text">
                <div className="status-label">Private Key Present</div>
                <div className="status-sublabel">Stored in IndexedDB</div>
              </div>
            </div>

            <div className="status-item">
              <div className="status-check">✔</div>
              <div className="status-text">
                <div className="status-label">Public Key Registered</div>
                <div className="status-sublabel">On your home instance</div>
              </div>
            </div>

            <div className="status-item">
              <div className="status-check">✔</div>
              <div className="status-text">
                <div className="status-label">Recovery File Exported</div>
                <div className="status-sublabel">Download available below</div>
              </div>
            </div>
          </div>
        </div>

        {/* Key Information Card */}
        <div className="key-info-card">
          <h3 className="card-title">Key Information</h3>

          {/* DID */}
          <div className="key-field">
            <label className="key-label">Decentralized Identifier (DID)</label>
            <div className="key-value-container">
              <code className="key-value">{did}</code>
              <button
                className={`copy-button ${copiedField === 'did' ? 'copied' : ''}`}
                onClick={() => handleCopyToClipboard(did, 'did')}
                title="Copy to clipboard"
              >
                {copiedField === 'did' ? '✓ Copied' : 'Copy'}
              </button>
            </div>
          </div>

          {/* Public Key Fingerprint */}
          <div className="key-field">
            <label className="key-label">Public Key Fingerprint (SHA-256)</label>
            <div className="key-value-container">
              <code className="key-value">{publicKeyFingerprint}</code>
              <button
                className={`copy-button ${copiedField === 'fingerprint' ? 'copied' : ''}`}
                onClick={() =>
                  handleCopyToClipboard(publicKeyFingerprint, 'fingerprint')
                }
                title="Copy to clipboard"
              >
                {copiedField === 'fingerprint' ? '✓ Copied' : 'Copy'}
              </button>
            </div>
          </div>

          {/* Recovery Code */}
          <div className="key-field">
            <label className="key-label">Recovery Code</label>
            <div className="recovery-code-section">
              {showRecoveryCode ? (
                <div className="key-value-container">
                  <code className="key-value recovery-code">
                    {recoveryCode}
                  </code>
                  <button
                    className={`copy-button ${copiedField === 'recovery' ? 'copied' : ''}`}
                    onClick={() =>
                      handleCopyToClipboard(recoveryCode, 'recovery')
                    }
                  >
                    {copiedField === 'recovery' ? '✓ Copied' : 'Copy'}
                  </button>
                </div>
              ) : (
                <button
                  className="reveal-button"
                  onClick={() => setShowRecoveryCode(true)}
                >
                  👁 Reveal Code
                </button>
              )}
            </div>
          </div>
        </div>

        {/* Actions Card */}
        <div className="actions-card">
          <h3 className="card-title">Key Actions</h3>

          <div className="actions-grid">
            <button
              className="action-btn primary"
              onClick={() => alert('Recovery file would download: recovery_' + Date.now() + '.json')}
            >
              📥 Export Recovery File
            </button>

            <button
              className="action-btn disabled"
              disabled
              title="Rotate Key - Sprint 2 feature"
            >
              🔄 Rotate Key
              <span className="disabled-label">Sprint 2</span>
            </button>

            <button
              className="action-btn disabled"
              disabled
              title="Revoke Key - Sprint 2 feature"
            >
              ✕ Revoke Key
              <span className="disabled-label">Sprint 2</span>
            </button>
          </div>
        </div>

        {/* Security Tips Card */}
        <div className="security-tips-card">
          <h3 className="card-title">Security Tips</h3>
          <ul className="tips-list">
            <li>Never share your recovery code with anyone</li>
            <li>Save your recovery file in a secure location</li>
            <li>Clear browser cache if using a shared device</li>
            <li>Your private key never leaves this device</li>
            <li>Losing your private key means losing permanent access</li>
          </ul>
        </div>
      </div>
    </div>
  );
}
