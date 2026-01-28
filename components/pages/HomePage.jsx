'use client';

import React, { useState } from 'react';
import '../styles/HomePage.css';

const POSTS = [
  {
    id: 1,
    author: 'alice@federate.tech',
    avatar: '👩',
    displayName: 'Alice Chen',
    handle: '@alice',
    timestamp: '2h ago',
    content: 'Just deployed a new federated instance! The decentralization is working beautifully. 🚀',
    replies: 12,
    boosts: 45,
    likes: 128,
    local: true,
    visibility: 'public'
  },
  {
    id: 2,
    author: 'bob@community.social',
    avatar: '👨',
    displayName: 'Bob Smith',
    handle: '@bob',
    timestamp: '4h ago',
    content: 'Love the transparency in this federated model. No hidden algorithms, just real human connections. 💙',
    replies: 8,
    boosts: 32,
    likes: 95,
    local: false,
    visibility: 'public'
  },
  {
    id: 3,
    author: 'charlie@tech-minds.io',
    avatar: '👨‍💻',
    displayName: 'Charlie Dev',
    handle: '@charlie',
    timestamp: '6h ago',
    content: 'Implemented end-to-end encryption for DMs. Your messages are truly private now. 🔐',
    replies: 24,
    boosts: 67,
    likes: 234,
    local: false,
    visibility: 'public'
  },
  {
    id: 4,
    author: 'diana@creator.hub',
    avatar: '🎨',
    displayName: 'Diana Art',
    handle: '@diana',
    timestamp: '8h ago',
    content: 'Creating art in a decentralized space feels different. My work is truly mine. No corporate overlords.',
    replies: 16,
    boosts: 88,
    likes: 312,
    local: false,
    visibility: 'followers'
  },
  {
    id: 5,
    author: 'eve@privacy-first.org',
    avatar: '🔒',
    displayName: 'Eve Security',
    handle: '@eve',
    timestamp: '10h ago',
    content: 'Audited the security model. Everything checks out. This is real encryption, not theater. ✓',
    replies: 5,
    boosts: 42,
    likes: 156,
    local: false,
    visibility: 'public'
  }
];

export default function HomePage({ onNavigate, userData, updateUserData, isDarkMode, toggleTheme }) {
  const [activeTab, setActiveTab] = useState('home');
  const [newPostText, setNewPostText] = useState('');
  const [posts, setPosts] = useState(POSTS);
  const [filteredPosts, setFilteredPosts] = useState(POSTS);

  const handlePostCreate = () => {
    if (newPostText.trim()) {
      const newPost = {
        id: posts.length + 1,
        author: `${userData.email}`,
        avatar: userData.avatar,
        displayName: userData.displayName,
        handle: `@${userData.username}`,
        timestamp: 'now',
        content: newPostText,
        replies: 0,
        boosts: 0,
        likes: 0,
        local: true,
        visibility: 'public'
      };
      setPosts([newPost, ...posts]);
      setNewPostText('');
    }
  };

  const getFilteredPosts = () => {
    if (activeTab === 'local') {
      return posts.filter(post => post.local);
    } else if (activeTab === 'federated') {
      return posts.filter(post => !post.local);
    }
    return posts;
  };

  return (
    <div className="home-container">
      {/* Top Navigation */}
      <nav className="home-nav">
        <div className="nav-left">
          <h1 className="nav-logo">🌐 FEDERATE</h1>
        </div>
        <div className="nav-center">
          <button 
            className="nav-item"
            onClick={() => setActiveTab('home')}
          >
            Home
          </button>
          <button 
            className="nav-item"
            onClick={() => setActiveTab('local')}
          >
            Local
          </button>
          <button 
            className="nav-item"
            onClick={() => setActiveTab('federated')}
          >
            Federated
          </button>
        </div>
        <div className="nav-right">
          <input 
            type="text" 
            placeholder="Search (disabled)" 
            disabled
            className="nav-search"
          />
          <button 
            className="nav-btn-profile"
            onClick={() => onNavigate('profile')}
          >
            👤 {userData.username}
          </button>
          <button 
            className="nav-btn-profile"
            onClick={toggleTheme}
            title={isDarkMode ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
            style={{
              marginLeft: '10px',
              padding: '8px 12px',
              background: isDarkMode ? 'rgba(0, 217, 255, 0.1)' : 'rgba(100, 100, 100, 0.1)',
              border: `1px solid ${isDarkMode ? '#00d9ff' : '#666'}`,
              color: isDarkMode ? '#00d9ff' : '#333',
              borderRadius: '6px',
              cursor: 'pointer'
            }}
          >
            {isDarkMode ? '🌙' : '☀️'}
          </button>
        </div>
      </nav>

      {/* Main Layout */}
      <div className="home-layout">
        {/* Left Sidebar */}
        <aside className="home-sidebar">
          <div className="sidebar-section">
            <h3 className="sidebar-title">Navigation</h3>
            <div className="sidebar-links">
              <button 
                className="sidebar-link active"
                onClick={() => setActiveTab('home')}
                style={{ textAlign: 'left', width: '100%' }}
              >
                <span className="icon">🏠</span>
                <span>Home</span>
              </button>
              <button 
                className="sidebar-link"
                onClick={() => onNavigate('dm')}
                style={{ textAlign: 'left', width: '100%' }}
              >
                <span className="icon">💬</span>
                <span>Messages 🔒</span>
              </button>
              <button 
                className="sidebar-link"
                onClick={() => onNavigate('security')}
                style={{ textAlign: 'left', width: '100%' }}
              >
                <span className="icon">🔐</span>
                <span>Security</span>
              </button>
            </div>
          </div>

          <div className="sidebar-section">
            <h3 className="sidebar-title">Your Profile</h3>
            <div className="sidebar-profile">
              <div className="profile-avatar">{userData.avatar}</div>
              <div className="profile-info">
                <p className="profile-name">{userData.displayName}</p>
                <p className="profile-handle">@{userData.username}@{userData.server}</p>
                <p className="profile-stats">
                  <strong>{userData.following}</strong> Following • <strong>{userData.followers}</strong> Followers
                </p>
              </div>
              <button 
                className="sidebar-btn"
                onClick={() => onNavigate('security')}
              >
                Settings ⚙️
              </button>
            </div>
          </div>

          <div className="sidebar-section">
            <h3 className="sidebar-title">Server Info</h3>
            <div className="sidebar-info">
              <div className="info-item">
                <span className="info-label">Server</span>
                <span className="info-value">federate.tech</span>
              </div>
              <div className="info-item">
                <span className="info-label">Reputation</span>
                <span className="info-value">🟢 Trusted</span>
              </div>
              <div className="info-item">
                <span className="info-label">Federation</span>
                <span className="info-value">🌐 Open</span>
              </div>
            </div>
          </div>
        </aside>

        {/* Main Feed */}
        <main className="home-feed">
          {/* Composer */}
          <div className="feed-composer">
            <div className="composer-header">
              <h2>What's happening? 🌐</h2>
            </div>
            <div className="composer-body">
              <textarea
                className="composer-textarea"
                placeholder="Share your thoughts with the federated network..."
                value={newPostText}
                onChange={(e) => setNewPostText(e.target.value)}
                maxLength="500"
              />
              <div className="composer-footer">
                <div className="composer-info">
                  <span className="char-count">
                    {newPostText.length}/500
                  </span>
                  <span className="visibility-icon">🌐 Public</span>
                </div>
                <div className="composer-actions">
                  <button 
                    className="composer-btn-media disabled"
                    disabled
                    title="Media upload - Sprint 2"
                  >
                    🖼️ Media
                  </button>
                  <button 
                    className={`composer-btn-post ${!newPostText.trim() ? 'disabled' : ''}`}
                    onClick={handlePostCreate}
                    disabled={!newPostText.trim()}
                  >
                    Post 🚀
                  </button>
                </div>
              </div>
            </div>
          </div>

          {/* Feed Divider */}
          <div className="feed-divider" />

          {/* Posts */}
          <div className="feed-posts">
            {getFilteredPosts().map(post => (
              <article key={post.id} className={`post ${post.local ? 'local' : 'remote'}`}>
                <div className="post-header">
                  <div className="post-author" style={{ cursor: 'pointer' }} onClick={() => onNavigate('profile')}>
                    <span className="post-avatar">{post.avatar}</span>
                    <div className="post-meta">
                      <div className="post-name-line">
                        <strong>{post.displayName}</strong>
                        <span className="post-handle">{post.handle}</span>
                        {post.local && <span className="post-badge local">🏠 Local</span>}
                        {!post.local && <span className="post-badge remote">🌐 Remote</span>}
                      </div>
                      <span className="post-time">{post.timestamp}</span>
                    </div>
                  </div>
                  {post.visibility === 'followers' && (
                    <span className="post-visibility">🔒 Followers Only</span>
                  )}
                </div>

                <div 
                  className="post-content"
                  style={{ cursor: 'pointer' }}
                  onClick={() => onNavigate('thread')}
                >
                  {post.content}
                </div>

                <div className="post-actions">
                  <button 
                    className="post-action"
                    onClick={() => onNavigate('thread')}
                  >
                    <span className="action-icon">💬</span>
                    <span className="action-count">{post.replies}</span>
                  </button>
                  <button className="post-action">
                    <span className="action-icon">🚀</span>
                    <span className="action-count">{post.boosts}</span>
                  </button>
                  <button className="post-action">
                    <span className="action-icon">❤️</span>
                    <span className="action-count">{post.likes}</span>
                  </button>
                  <button className="post-action post-action-delete">
                    <span className="action-icon">⋯</span>
                  </button>
                </div>
              </article>
            ))}
          </div>
        </main>

        {/* Right Sidebar */}
        <aside className="home-trends">
          <div className="trends-section">
            <h3 className="trends-title">🔥 Trending Topics</h3>
            <div className="trends-list">
              <a href="#" className="trend-item">
                <div className="trend-name">#Decentralization</div>
                <div className="trend-count">2.4K posts</div>
              </a>
              <a href="#" className="trend-item">
                <div className="trend-name">#Federation</div>
                <div className="trend-count">1.8K posts</div>
              </a>
              <a href="#" className="trend-item">
                <div className="trend-name">#PrivacyFirst</div>
                <div className="trend-count">942 posts</div>
              </a>
              <a href="#" className="trend-item">
                <div className="trend-name">#OpenSource</div>
                <div className="trend-count">3.1K posts</div>
              </a>
            </div>
          </div>

          <div className="trends-section">
            <h3 className="trends-title">ℹ️ About This Network</h3>
            <p className="trends-description">
              This is a decentralized social network powered by federation. Your identity is your own, your server is your choice, and your data is encrypted.
            </p>
          </div>

          <div className="trends-section">
            <h3 className="trends-title">⚙️ Admin Panel</h3>
            <button 
              onClick={() => onNavigate('moderation')}
              style={{
                width: '100%',
                padding: '10px',
                background: 'rgba(0, 217, 255, 0.1)',
                border: '1px solid #00d9ff',
                color: '#00d9ff',
                borderRadius: '6px',
                cursor: 'pointer',
                marginBottom: '8px',
                fontWeight: '600',
                transition: 'all 0.3s ease'
              }}
              className="trend-item"
            >
              📋 Moderation Queue
            </button>
            <button 
              onClick={() => onNavigate('federation')}
              style={{
                width: '100%',
                padding: '10px',
                background: 'rgba(0, 217, 255, 0.1)',
                border: '1px solid #00d9ff',
                color: '#00d9ff',
                borderRadius: '6px',
                cursor: 'pointer',
                fontWeight: '600',
                transition: 'all 0.3s ease'
              }}
              className="trend-item"
            >
              🌐 Federation Inspector
            </button>
          </div>

          <div className="trends-section">
            <h3 className="trends-title">🚀 Future Features (Disabled)</h3>
            <ul className="features-list">
              <li>📁 Media Upload - Sprint 2</li>
              <li>👥 Custom Circles - Sprint 2</li>
              <li>🔍 Search - Sprint 2</li>
              <li>📊 Federation Graph - Sprint 3</li>
              <li>⭐ Reputation Scoring - Sprint 3</li>
            </ul>
          </div>
        </aside>
      </div>
    </div>
  );
}
