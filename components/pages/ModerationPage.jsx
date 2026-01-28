'use client';

import { useState } from 'react';
import '../styles/ModerationPage.css';

export default function ModerationPage({ onNavigate, isDarkMode, toggleTheme }) {
  const [filterType, setFilterType] = useState('all');
  const [queue, setQueue] = useState([
    {
      id: 1,
      preview: 'Buy crypto now! Guaranteed 1000% returns!!!',
      author: '@spam_bot',
      server: 'evil.net',
      isFederated: true,
      reason: 'Spam',
      timestamp: '2 minutes ago',
      actions: ['Remove', 'Block User'],
    },
    {
      id: 2,
      preview: 'This user is a complete idiot and should...',
      author: '@angry_user',
      server: 'social.example.net',
      isFederated: false,
      reason: 'Harassment',
      timestamp: '5 minutes ago',
      actions: ['Warn', 'Mute', 'Remove'],
    },
    {
      id: 3,
      preview: 'Check out my adult content store...',
      author: '@merchant123',
      server: 'commerce.net',
      isFederated: true,
      reason: 'Spam',
      timestamp: '12 minutes ago',
      actions: ['Remove', 'Block Domain'],
    },
    {
      id: 4,
      preview: 'Hate groups are actually okay because...',
      author: '@extremist',
      server: 'hate.net',
      isFederated: true,
      reason: 'Hate Speech',
      timestamp: '20 minutes ago',
      actions: ['Remove', 'Block Domain'],
    },
    {
      id: 5,
      preview: 'Can someone help me with this math problem?',
      author: '@student',
      server: 'social.example.net',
      isFederated: false,
      reason: 'Reported by User',
      timestamp: '1 hour ago',
      actions: ['Approve', 'Dismiss'],
    },
  ]);

  const handleAction = (id, action) => {
    if (action === 'Remove') {
      setQueue(queue.filter((item) => item.id !== id));
      alert(`Removed post ${id}`);
    } else if (action === 'Block Domain') {
      setQueue(queue.filter((item) => item.id !== id));
      alert(`Domain ${queue.find((i) => i.id === id)?.server} blocked`);
    } else if (action === 'Warn') {
      setQueue(queue.filter((item) => item.id !== id));
      alert(`User warned for post ${id}`);
    } else if (action === 'Approve') {
      setQueue(queue.filter((item) => item.id !== id));
      alert(`Post ${id} approved`);
    }
  };

  const filteredQueue = queue.filter((item) => {
    if (filterType === 'spam') return item.reason === 'Spam';
    if (filterType === 'harassment') return item.reason === 'Harassment';
    if (filterType === 'federated') return item.isFederated;
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
          <button onClick={() => onNavigate('profile')} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>👤 Profile</button>
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

        {/* Moderation Queue Table */}
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
                  <div className="preview-text">{item.preview}</div>
                </div>
                <div className="col-user">{item.author}</div>
                <div className="col-server">
                  {item.server}
                  {item.isFederated && (
                    <span className="federated-badge">🌐</span>
                  )}
                </div>
                <div className="col-reason">
                  <span className={`reason-tag ${item.reason.toLowerCase().replace(' ', '-')}`}>
                    {item.reason}
                  </span>
                </div>
                <div className="col-action">
                  <div className="action-buttons">
                    {item.actions.map((action) => (
                      <button
                        key={action}
                        className={`action-btn ${action.toLowerCase().replace(' ', '-')}`}
                        onClick={() => handleAction(item.id, action)}
                      >
                        {action}
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            ))
          ) : (
            <div className="empty-queue">
              <p>No items in queue</p>
            </div>
          )}
        </div>

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
