'use client';

import React from 'react';
import { useTheme } from '@/components/ui/theme-provider';
import '../styles/LandingPage.css';

export default function LandingPage({ onNavigate }) {
  const { theme, toggleTheme } = useTheme();
  const isDarkMode = theme === 'dark';
  return (
    <div className="landing-container">
      {/* Navigation Bar */}
      <nav className="landing-nav">
        <div className="landing-nav-left">
          <span className="landing-logo">🌐 SPLITTER</span>
        </div>
        <div className="landing-nav-right">
          <button
            className="landing-nav-btn"
            onClick={() => onNavigate('instances')}
          >
            Explore Network
          </button>
          <button
            className="landing-nav-btn"
            onClick={() => onNavigate('login')}
          >
            Login
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
              fontWeight: '600',
              marginRight: '10px'
            }}
          >
            {isDarkMode ? '🌙' : '☀️'}
          </button>
          <button
            className="landing-nav-btn landing-nav-btn-primary"
            onClick={() => onNavigate('signup')}
          >
            Sign Up
          </button>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="landing-hero">
        <div className="landing-hero-content">
          <h1 className="landing-hero-title">SPLITTER</h1>
          <p className="landing-hero-subtitle">
            Your Identity. Your Server. Your Network.
          </p>
          <p className="landing-hero-description">
            A decentralized social platform where you own your identity,
            choose your server, and connect across a federated network.
          </p>
        </div>

        {/* Security Banner */}
        <div className="landing-banner">
          <span className="landing-banner-icon">🔒</span>
          <p className="landing-banner-text">
            This platform uses decentralized identity. Your private keys never leave your device.
          </p>
        </div>
      </section>

      {/* Features Grid */}
      <section className="landing-features">
        <h2 className="landing-section-title">Why Federate?</h2>
        <div className="landing-features-grid">
          <div className="landing-feature-card">
            <div className="landing-feature-icon">🔐</div>
            <h3>Own Your Keys</h3>
            <p>Generate cryptographic identity stored securely on your device</p>
          </div>
          <div className="landing-feature-card">
            <div className="landing-feature-icon">🧭</div>
            <h3>Choose Server</h3>
            <p>Join communities with different moderation policies and values</p>
          </div>
          <div className="landing-feature-card">
            <div className="landing-feature-icon">🌐</div>
            <h3>True Interop</h3>
            <p>Connect across federated networks and reach everyone, everywhere</p>
          </div>
        </div>
      </section>

      {/* Federation Explanation */}
      <section className="landing-federation">
        <h2 className="landing-section-title">How Federation Works</h2>
        <div className="landing-federation-content">
          <div className="landing-federation-item">
            <div className="landing-federation-step">1</div>
            <h4>Create Identity</h4>
            <p>Generate a decentralized identifier (DID) using WebCrypto</p>
          </div>
          <div className="landing-federation-item">
            <div className="landing-federation-step">2</div>
            <h4>Join Server</h4>
            <p>Choose a server community aligned with your values</p>
          </div>
          <div className="landing-federation-item">
            <div className="landing-federation-step">3</div>
            <h4>Connect</h4>
            <p>Interact with users across all federated servers</p>
          </div>
          <div className="landing-federation-item">
            <div className="landing-federation-step">4</div>
            <h4>Own Data</h4>
            <p>Your content and keys remain yours to control</p>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="landing-cta">
        <h2>Ready to join the federated network?</h2>
        <div className="landing-cta-buttons">
          <button
            className="landing-btn-primary"
            onClick={() => onNavigate('instances')}
          >
            Explore Public Network →
          </button>
          <button
            className="landing-btn-secondary"
            onClick={() => onNavigate('signup')}
          >
            Join a Server
          </button>
        </div>
      </section>

      {/* Footer */}
      <footer className="landing-footer">
        <p>&copy; 2024 Federate - Your Identity. Your Network.</p>
        <div className="landing-footer-links">
          <a href="#docs">Documentation</a>
          <a href="#privacy">Privacy</a>
          <a href="#terms">Terms</a>
        </div>
      </footer>
    </div>
  );
}
