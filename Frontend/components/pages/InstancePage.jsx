'use client';

import React, { useState } from 'react';
import { useTheme } from '@/components/ui/theme-provider';
import '../styles/InstancePage.css';

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
    region: 'Delhi'
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
    region: 'Karnataka'
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
    region: 'Maharashtra'
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
    region: 'West Bengal'
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
    region: 'Telangana'
  },
  {
    id: 6,
    name: 'privacy-guard.in',
    category: 'Privacy & Security',
    users: 1200,
    federation: 'Open',
    moderation: 'Strict',
    reputation: 'trusted',
    description: 'Privacy-focused server for India - secure communications and encryption',
    region: 'Pan-India'
  }
];

const REGIONS = ['All', 'Delhi', 'Karnataka', 'Maharashtra', 'West Bengal', 'Telangana', 'Pan-India'];
const MODERATION_LEVELS = ['All', 'Strict', 'Moderate', 'Lenient'];

export default function InstancePage({ onNavigate }) {
  const { theme, toggleTheme } = useTheme();
  const isDarkMode = theme === 'dark';
  const [selectedFilter, setSelectedFilter] = useState('all');
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedRegion, setSelectedRegion] = useState('All');
  const [selectedModeration, setSelectedModeration] = useState('All');

  const filteredServers = SERVERS.filter(server => {
    const matchesSearch = server.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      server.category.toLowerCase().includes(searchTerm.toLowerCase()) ||
      server.description.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesRegion = selectedRegion === 'All' || server.region === selectedRegion;
    const matchesModeration = selectedModeration === 'All' || server.moderation === selectedModeration;

    return matchesSearch && matchesRegion && matchesModeration;
  });

  return (
    <div className="instance-container">
      {/* Header */}
      <div className="instance-header">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '2rem' }}>
          <button
            className="instance-back-btn"
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
        <h1 className="instance-title">🧭 Join a Federated Server</h1>
        <p className="instance-subtitle">
          Choose a server community aligned with your values and interests
        </p>
      </div>

      {/* Search & Filter */}
      <div className="instance-controls">
        <div className="instance-search">
          <span className="search-icon">🔍</span>
          <input
            type="text"
            placeholder="Search servers..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="search-input"
          />
        </div>

        <div className="instance-filters">
          <div className="filter-group">
            <label className="filter-label">Region:</label>
            <select
              value={selectedRegion}
              onChange={(e) => setSelectedRegion(e.target.value)}
              className="filter-select"
            >
              {REGIONS.map(region => (
                <option key={region} value={region}>{region}</option>
              ))}
            </select>
          </div>

          <div className="filter-group">
            <label className="filter-label">Moderation:</label>
            <select
              value={selectedModeration}
              onChange={(e) => setSelectedModeration(e.target.value)}
              className="filter-select"
            >
              {MODERATION_LEVELS.map(level => (
                <option key={level} value={level}>{level}</option>
              ))}
            </select>
          </div>

          <button
            className="filter-chip disabled"
            disabled
            title="Reputation filtering - Sprint 3"
          >
            ⭐ Reputation (Sprint 3)
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
      </div>

      {/* Servers Grid */}
      <div className="instance-grid">
        {filteredServers.map(server => (
          <div
            key={server.id}
            className={`instance-card ${server.blocked ? 'blocked' : ''}`}
          >
            <div className="instance-card-header">
              <h3 className="instance-card-name">
                {server.name}
                {server.blocked && ' 🛑'}
              </h3>
              <span className="instance-card-federation">
                {server.federation === 'Open' ? '🌐' : '🛑'} {server.federation}
              </span>
            </div>

            <p className="instance-card-category">{server.category}</p>
            <p className="instance-card-description">{server.description}</p>

            <div className="instance-card-stats">
              <div className="stat">
                <span className="stat-label">Users</span>
                <span className="stat-value">{server.users.toLocaleString()}</span>
              </div>
              <div className="stat">
                <span className="stat-label">Region</span>
                <span className="stat-value">{server.region}</span>
              </div>
              <div className="stat">
                <span className="stat-label">Moderation</span>
                <span className="stat-value">{server.moderation}</span>
              </div>
            </div>

            {server.reputation === 'trusted' && (
              <div className="reputation-badge">
                <span>🟢 Trusted Network</span>
              </div>
            )}

            {server.blocked && (
              <div className="blocked-badge">
                <span>⚠️ Blocked by Admin</span>
              </div>
            )}

            <button
              className={`instance-card-btn ${server.blocked ? 'blocked-btn' : ''}`}
              onClick={() => !server.blocked && onNavigate('signup')}
              disabled={server.blocked}
            >
              {server.blocked ? 'Blocked' : 'Join Server'}
            </button>
          </div>
        ))}
      </div>

      {filteredServers.length === 0 && (
        <div className="instance-empty">
          <p>No servers found matching your criteria</p>
        </div>
      )}
    </div>
  );
}
