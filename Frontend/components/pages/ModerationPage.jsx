'use client';

import { useState, useEffect } from 'react';
import { useTheme } from '@/components/ui/theme-provider';
import '../styles/ModerationPage.css';
import { adminApi } from '@/lib/api';

export default function ModerationPage({ onNavigate, userData }) {
  const { theme, toggleTheme } = useTheme();
  const isDarkMode = theme === 'dark';
  const [filterType, setFilterType] = useState('all');
  const [queue, setQueue] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(null);

  // Check permissions
  if (userData?.role !== 'admin' && userData?.role !== 'moderator') {
    return (
      <div style={{
        minHeight: '100vh',
        background: '#0f0f1a',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        color: '#fff'
      }}>
        <h1 style={{ color: '#ff4444', marginBottom: '16px' }}>⛔ Access Denied</h1>
        <p style={{ color: '#666', marginBottom: '24px' }}>You need moderator privileges to access this page.</p>
        <button
          onClick={() => onNavigate('home')}
          style={{
            padding: '12px 24px',
            background: 'rgba(0,217,255,0.2)',
            border: '1px solid #00d9ff',
            color: '#00d9ff',
            borderRadius: '8px',
            cursor: 'pointer'
          }}
        >
          ← Back to Home
        </button>
      </div>
    );
  }

  useEffect(() => {
    fetchModerationQueue();
  }, []);

  const fetchModerationQueue = async () => {
    setIsLoading(true);
    try {
      const result = await adminApi.getModerationQueue();
      setQueue(result.items || []);
    } catch (err) {
      console.error('Failed to fetch moderation queue:', err);
      // Fall back to empty queue on error
      setQueue([]);
    } finally {
      setIsLoading(false);
    }
  };

  const handleAction = async (id, action, item) => {
    setActionLoading(id);
    try {
      if (action === 'Remove') {
        await adminApi.removeContent(id);
        setQueue(queue.filter((q) => q.id !== id));
        alert(`Content removed successfully`);
      } else if (action === 'Block Domain') {
        await adminApi.blockDomain(item.server);
        setQueue(queue.filter((q) => q.id !== id));
        alert(`Domain ${item.server} blocked`);
      } else if (action === 'Warn') {
        await adminApi.warnUser(item.author_id, item.reason);
        setQueue(queue.filter((q) => q.id !== id));
        alert(`User warned for: ${item.reason}`);
      } else if (action === 'Approve') {
        await adminApi.approveContent(id);
        setQueue(queue.filter((q) => q.id !== id));
        alert(`Content approved`);
      } else if (action === 'Mute') {
        await adminApi.suspendUser(item.author_id);
        setQueue(queue.filter((q) => q.id !== id));
        alert(`User muted`);
      } else if (action === 'Dismiss') {
        await adminApi.approveContent(id);
        setQueue(queue.filter((q) => q.id !== id));
        alert(`Report dismissed`);
      } else if (action === 'Block User') {
        await adminApi.suspendUser(item.author_id);
        setQueue(queue.filter((q) => q.id !== id));
        alert(`User blocked`);
      }
    } catch (err) {
      alert(`Action failed: ${err.message}`);
    } finally {
      setActionLoading(null);
    }
  };

  const getActionsForItem = (item) => {
    const reason = item.reason?.toLowerCase() || '';
    if (reason.includes('spam')) {
      return ['Remove', 'Block User'];
    } else if (reason.includes('harassment')) {
      return ['Warn', 'Mute', 'Remove'];
    } else if (reason.includes('hate')) {
      return ['Remove', 'Block Domain'];
    } else if (item.isFederated) {
      return ['Remove', 'Block Domain'];
    } else {
      return ['Approve', 'Dismiss', 'Remove'];
    }
  };

  const filteredQueue = queue.filter((item) => {
    if (filterType === 'spam') return item.reason?.toLowerCase().includes('spam');
    if (filterType === 'harassment') return item.reason?.toLowerCase().includes('harassment');
    if (filterType === 'federated') return item.isFederated || item.is_federated;
    return true;
  });

  return (
    <div className="moderation-container">
      {/* Navigation */}
      <div className="moderation-navbar">
        <button
          className="nav-button back-button"
          onClick={() => onNavigate('home')}
        >
          ← Back
        </button>
        <h1 className="navbar-title">Moderation Panel</h1>
        <div style={{ marginLeft: 'auto', display: 'flex', gap: '10px' }}>
          <button onClick={() => onNavigate('federation')} style={{ padding: '8px 12px', background: 'rgba(255,0,110,0.1)', border: '1px solid #ff006e', color: '#ff006e', borderRadius: '6px', cursor: 'pointer' }}>🌐 Federation</button>
          <button onClick={() => onNavigate('profile')} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>Profile</button>
          <button onClick={toggleTheme} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>{isDarkMode ? '🌙' : '☀️'}</button>
        </div>
      </div>

      <div className="moderation-content">
        {/* Header Section */}
        <div className="moderation-header">
          <div className="header-info">
            <h2>Content Moderation Queue</h2>
            <p>{filteredQueue.length} items in queue</p>
          </div>
          <button
            onClick={fetchModerationQueue}
            disabled={isLoading}
            style={{
              padding: '8px 16px',
              background: 'rgba(0,217,255,0.1)',
              border: '1px solid #00d9ff',
              color: '#00d9ff',
              borderRadius: '6px',
              cursor: isLoading ? 'not-allowed' : 'pointer'
            }}
          >
            🔄 Refresh
          </button>
        </div>

        {/* Filter Buttons */}
        <div className="filter-chips">
          <button
            className={`chip ${filterType === 'all' ? 'active' : ''}`}
            onClick={() => setFilterType('all')}
          >
            All
          </button>
          <button
            className={`chip ${filterType === 'spam' ? 'active' : ''}`}
            onClick={() => setFilterType('spam')}
          >
            Spam
          </button>
          <button
            className={`chip ${filterType === 'harassment' ? 'active' : ''}`}
            onClick={() => setFilterType('harassment')}
          >
            Harassment
          </button>
          <button
            className={`chip ${filterType === 'federated' ? 'active' : ''}`}
            onClick={() => setFilterType('federated')}
          >
            Federated Only 🌐
          </button>
        </div>

        {/* Loading State */}
        {isLoading ? (
          <div style={{ textAlign: 'center', padding: '60px', color: '#666' }}>
            Loading moderation queue...
          </div>
        ) : (
          /* Moderation Queue Table */
          <div className="queue-table">
            <div className="table-header">
              <div className="col-preview">Preview</div>
              <div className="col-user">User</div>
              <div className="col-server">Server</div>
              <div className="col-reason">Reason</div>
              <div className="col-action">Action</div>
            </div>

            {filteredQueue.length > 0 ? (
              filteredQueue.map((item) => (
                <div key={item.id} className="table-row">
                  <div className="col-preview">
                    <div className="preview-text">{item.preview || item.content}</div>
                  </div>
                  <div className="col-user">{item.author || item.username || '@unknown'}</div>
                  <div className="col-server">
                    {item.server || item.instance_domain || 'local'}
                    {(item.isFederated || item.is_federated) && (
                      <span className="federated-badge">🌐</span>
                    )}
                  </div>
                  <div className="col-reason">
                    <span className={`reason-tag ${(item.reason || 'reported').toLowerCase().replace(' ', '-')}`}>
                      {item.reason || 'Reported by User'}
                    </span>
                  </div>
                  <div className="col-action">
                    <div className="action-buttons">
                      {getActionsForItem(item).map((action) => (
                        <button
                          key={action}
                          className={`action-btn ${action.toLowerCase().replace(' ', '-')}`}
                          onClick={() => handleAction(item.id, action, item)}
                          disabled={actionLoading === item.id}
                          style={{ opacity: actionLoading === item.id ? 0.5 : 1 }}
                        >
                          {actionLoading === item.id ? '...' : action}
                        </button>
                      ))}
                    </div>
                  </div>
                </div>
              ))
            ) : (
              <div className="empty-queue">
                <div style={{ fontSize: '48px', marginBottom: '16px' }}>✨</div>
                <p>No items in queue</p>
                <p style={{ color: '#666', fontSize: '14px' }}>All caught up! Great work.</p>
              </div>
            )}
          </div>
        )}

        {/* Moderation Notes */}
        <div className="moderation-notes">
          <h3>Moderation Guidelines</h3>
          <ul>
            <li>
              <strong>Local posts:</strong> You can remove, warn, or mute users
            </li>
            <li>
              <strong>Federated posts:</strong> Can only be removed from your
              instance timeline (notify remote server in Sprint 2)
            </li>
            <li>
              <strong>Domain blocking:</strong> Prevents all content from that
              server
            </li>
            <li>
              <strong>User muting:</strong> Hides their posts from all timelines
            </li>
          </ul>
        </div>
      </div>
    </div>
  );
}
