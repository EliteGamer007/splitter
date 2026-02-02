'use client';

import { useState } from 'react';
import { useTheme } from '@/components/ui/theme-provider';
import '../styles/FederationPage.css';

export default function FederationPage({ onNavigate }) {
  const { theme, toggleTheme } = useTheme();
  const isDarkMode = theme === 'dark';
  const [connectedServers] = useState([
    {
      id: 1,
      domain: 'node1.social',
      status: 'healthy',
      reputation: 'Trusted',
      lastSeen: '2 minutes ago',
      activities: '142/min',
    },
    {
      id: 2,
      domain: 'federated.example.net',
      status: 'healthy',
      reputation: 'Trusted',
      lastSeen: '1 minute ago',
      activities: '89/min',
    },
    {
      id: 3,
      domain: 'crypto.social',
      status: 'degraded',
      reputation: 'Moderate',
      lastSeen: '5 minutes ago',
      activities: '23/min',
    },
    {
      id: 4,
      domain: 'privacy.net',
      status: 'healthy',
      reputation: 'Trusted',
      lastSeen: '1 minute ago',
      activities: '67/min',
    },
    {
      id: 5,
      domain: 'evil.net',
      status: 'blocked',
      reputation: 'Blocked',
      lastSeen: '—',
      activities: '0/min',
    },
  ]);

  const getStatusColor = (status) => {
    switch (status) {
      case 'healthy':
        return '#00ff88';
      case 'degraded':
        return '#ffd700';
      case 'blocked':
        return '#ff4444';
      default:
        return '#888';
    }
  };

  const getStatusBadge = (status) => {
    switch (status) {
      case 'healthy':
        return '🟢';
      case 'degraded':
        return '🟡';
      case 'blocked':
        return '🔴';
      default:
        return '⚪';
    }
  };

  return (
    <div className="federation-container">
      {/* Navigation */}
      <div className="federation-navbar">
        <button
          className="nav-button back-button"
          onClick={() => onNavigate('home')}
        >
          ← Back
        </button>
        <h1 className="navbar-title">Federation Inspector</h1>
        <div style={{ marginLeft: 'auto', display: 'flex', gap: '10px' }}>
          <button onClick={() => onNavigate('moderation')} style={{ padding: '8px 12px', background: 'rgba(255,0,110,0.1)', border: '1px solid #ff006e', color: '#ff006e', borderRadius: '6px', cursor: 'pointer' }}>📋 Moderation</button>
          <button onClick={() => onNavigate('profile')} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>Profile</button>
          <button onClick={toggleTheme} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>{isDarkMode ? '🌙' : '☀️'}</button>
        </div>
      </div>

      <div className="federation-content">
        {/* Health Metrics */}
        <div className="health-section">
          <h2 className="section-title">Federation Health</h2>

          <div className="metrics-grid">
            <div className="metric-card">
              <div className="metric-icon">📥</div>
              <div className="metric-info">
                <div className="metric-label">Incoming Activities</div>
                <div className="metric-value">14/min</div>
              </div>
            </div>

            <div className="metric-card">
              <div className="metric-icon">📤</div>
              <div className="metric-info">
                <div className="metric-label">Outgoing Activities</div>
                <div className="metric-value">9/min</div>
              </div>
            </div>

            <div className="metric-card">
              <div className="metric-icon">✔</div>
              <div className="metric-info">
                <div className="metric-label">Signature Validation</div>
                <div className="metric-value">100%</div>
              </div>
            </div>

            <div className="metric-card">
              <div className="metric-icon">⏳</div>
              <div className="metric-info">
                <div className="metric-label">Retry Queue</div>
                <div className="metric-value">2 pending</div>
              </div>
            </div>
          </div>
        </div>

        {/* Activity Chart (Visualization) */}
        <div className="activity-chart">
          <h3>Activity Over Time</h3>
          <div className="chart-placeholder">
            <div className="chart-bar" style={{ height: '60%' }}></div>
            <div className="chart-bar" style={{ height: '75%' }}></div>
            <div className="chart-bar" style={{ height: '45%' }}></div>
            <div className="chart-bar" style={{ height: '85%' }}></div>
            <div className="chart-bar" style={{ height: '50%' }}></div>
            <div className="chart-bar" style={{ height: '70%' }}></div>
            <div className="chart-bar" style={{ height: '65%' }}></div>
          </div>
          <div className="chart-labels">
            <span>Last 7 minutes</span>
          </div>
        </div>

        {/* Connected Servers Table */}
        <div className="servers-section">
          <h2 className="section-title">Connected Servers</h2>

          <div className="servers-table">
            <div className="table-header">
              <div className="col-domain">Domain</div>
              <div className="col-status">Status</div>
              <div className="col-reputation">Reputation</div>
              <div className="col-lastseen">Last Seen</div>
              <div className="col-activities">Activities</div>
            </div>

            {connectedServers.map((server) => (
              <div key={server.id} className="table-row">
                <div className="col-domain">
                  <span className="domain-name">{server.domain}</span>
                </div>
                <div className="col-status">
                  <span
                    className={`status-badge ${server.status}`}
                    style={{
                      borderColor: getStatusColor(server.status),
                    }}
                  >
                    {getStatusBadge(server.status)} {server.status}
                  </span>
                </div>
                <div className="col-reputation">
                  <span className={`reputation-tag ${server.reputation.toLowerCase().replace(' ', '-')}`}>
                    {server.reputation}
                  </span>
                </div>
                <div className="col-lastseen">{server.lastSeen}</div>
                <div className="col-activities">{server.activities}</div>
              </div>
            ))}
          </div>
        </div>

        {/* Legend */}
        <div className="legend-section">
          <h3>Status Indicators</h3>
          <div className="legend-items">
            <div className="legend-item">
              <span className="legend-color healthy">🟢</span>
              <span>Healthy - All systems operational</span>
            </div>
            <div className="legend-item">
              <span className="legend-color degraded">🟡</span>
              <span>Degraded - Slow response or high errors</span>
            </div>
            <div className="legend-item">
              <span className="legend-color blocked">🔴</span>
              <span>Blocked - No communication allowed</span>
            </div>
          </div>
        </div>

        {/* Advanced Options */}
        <div className="advanced-section">
          <h3>Advanced Options</h3>
          <div className="advanced-buttons">
            <button className="advanced-btn" disabled title="Sprint 2 feature">
              🔄 Resync Federation
              <span className="disabled-label">Sprint 2</span>
            </button>
            <button className="advanced-btn" disabled title="Sprint 2 feature">
              📋 Export Activity Log
              <span className="disabled-label">Sprint 2</span>
            </button>
            <button className="advanced-btn" disabled title="Sprint 3 feature">
              📊 Detailed Analytics
              <span className="disabled-label">Sprint 3</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
