'use client';

import React, { useState, useEffect } from 'react';
import '../styles/HomePage.css';
import { postApi, interactionApi, adminApi, searchApi, messageApi } from '@/lib/api';

// Sample posts for demo when no backend posts available
const SAMPLE_POSTS = [
  {
    id: 'sample-1',
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
    id: 'sample-2',
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
    id: 'sample-3',
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
  }
];

export default function HomePage({ onNavigate, userData, updateUserData, isDarkMode, toggleTheme, handleLogout }) {
  const [activeTab, setActiveTab] = useState('home');
  const [newPostText, setNewPostText] = useState('');
  const [posts, setPosts] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isPosting, setIsPosting] = useState(false);
  const [error, setError] = useState(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [showSearchResults, setShowSearchResults] = useState(false);
  const [searchResults, setSearchResults] = useState([]);
  const [isSearching, setIsSearching] = useState(false);

  // Fetch posts on mount
  useEffect(() => {
    fetchPosts();
  }, []);

  // Search users
  const handleSearch = async () => {
    if (searchQuery.length < 2) return;
    
    setIsSearching(true);
    try {
      const result = await searchApi.searchUsers(searchQuery);
      setSearchResults(result.users || []);
      setShowSearchResults(true);
    } catch (err) {
      console.error('Search failed:', err);
      setSearchResults([]);
    } finally {
      setIsSearching(false);
    }
  };

  // Debounced search
  useEffect(() => {
    if (searchQuery.length >= 2) {
      const timer = setTimeout(handleSearch, 300);
      return () => clearTimeout(timer);
    } else {
      setShowSearchResults(false);
      setSearchResults([]);
    }
  }, [searchQuery]);

  // Start DM with user
  const startDMWithUser = async (user) => {
    try {
      await messageApi.startConversation(user.id);
      setShowSearchResults(false);
      setSearchQuery('');
      onNavigate('dm', { selectedUser: user });
    } catch (err) {
      console.error('Failed to start conversation:', err);
      alert('Failed to start conversation: ' + err.message);
    }
  };

  const fetchPosts = async () => {
    setIsLoading(true);
    setError(null);
    try {
      // Try authenticated feed first, fall back to public feed
      let feedPosts;
      const token = typeof window !== 'undefined' ? localStorage.getItem('jwt_token') : null;
      
      if (token) {
        try {
          feedPosts = await postApi.getFeed(20, 0);
        } catch (authErr) {
          // Auth feed failed, try public
          feedPosts = await postApi.getPublicFeed(20, 0);
        }
      } else {
        // No token, use public feed
        feedPosts = await postApi.getPublicFeed(20, 0);
      }
      
      if (feedPosts && feedPosts.length > 0) {
        // Transform API posts to display format
        const transformedPosts = feedPosts.map(post => ({
          id: post.id,
          author: post.username ? `${post.username}@localhost` : `${post.author_did?.split(':').pop() || 'unknown'}@local`,
          avatar: post.avatar_url || '👤',
          displayName: post.username || post.author_did?.split(':').pop() || 'Unknown',
          handle: `@${post.username || post.author_did?.split(':').pop() || 'unknown'}`,
          timestamp: formatTimestamp(post.created_at),
          content: post.content,
          replies: post.reply_count || 0,
          boosts: post.repost_count || 0,
          likes: post.like_count || 0,
          local: true,
          visibility: post.visibility || 'public',
          liked: post.liked || false,
          reposted: post.reposted || false,
          bookmarked: post.bookmarked || false
        }));
        setPosts(transformedPosts);
      } else {
        // No posts from API, show samples
        setPosts(SAMPLE_POSTS);
      }
    } catch (err) {
      console.error('Failed to fetch posts:', err);
      // Fall back to sample posts on error
      setPosts(SAMPLE_POSTS);
    } finally {
      setIsLoading(false);
    }
  };

  const formatTimestamp = (isoString) => {
    if (!isoString) return 'unknown';
    const date = new Date(isoString);
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);
    
    if (diffMins < 1) return 'now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    return `${diffDays}d ago`;
  };

  const handlePostCreate = async () => {
    if (!newPostText.trim()) return;

    setIsPosting(true);
    try {
      const newPost = await postApi.createPost(newPostText);
      
      // Add to top of posts list
      const transformedPost = {
        id: newPost.id,
        author: `${userData.username}@${userData.server}`,
        avatar: userData.avatar,
        displayName: userData.displayName,
        handle: `@${userData.username}`,
        timestamp: 'now',
        content: newPost.content,
        replies: 0,
        boosts: 0,
        likes: 0,
        local: true,
        visibility: 'public'
      };
      
      setPosts(prev => [transformedPost, ...prev]);
      setNewPostText('');
    } catch (err) {
      setError('Failed to create post: ' + err.message);
    } finally {
      setIsPosting(false);
    }
  };

  const handleLike = async (postId) => {
    try {
      const post = posts.find(p => p.id === postId);
      if (post?.liked) {
        await interactionApi.unlikePost(postId);
        setPosts(prev => prev.map(p => 
          p.id === postId ? { ...p, liked: false, likes: p.likes - 1 } : p
        ));
      } else {
        await interactionApi.likePost(postId);
        setPosts(prev => prev.map(p => 
          p.id === postId ? { ...p, liked: true, likes: p.likes + 1 } : p
        ));
      }
    } catch (err) {
      console.error('Like failed:', err);
    }
  };

  const handleRepost = async (postId) => {
    try {
      const post = posts.find(p => p.id === postId);
      if (post?.reposted) {
        await interactionApi.unrepostPost(postId);
        setPosts(prev => prev.map(p => 
          p.id === postId ? { ...p, reposted: false, boosts: p.boosts - 1 } : p
        ));
      } else {
        await interactionApi.repostPost(postId);
        setPosts(prev => prev.map(p => 
          p.id === postId ? { ...p, reposted: true, boosts: p.boosts + 1 } : p
        ));
      }
    } catch (err) {
      console.error('Repost failed:', err);
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
          <h1 className="nav-logo">🌐 SPLITTER</h1>
        </div>
        <div className="nav-center">
          <button 
            className={`nav-item ${activeTab === 'home' ? 'active' : ''}`}
            onClick={() => setActiveTab('home')}
          >
            Home
          </button>
          <button 
            className={`nav-item ${activeTab === 'local' ? 'active' : ''}`}
            onClick={() => setActiveTab('local')}
          >
            Local
          </button>
          <button 
            className={`nav-item ${activeTab === 'federated' ? 'active' : ''}`}
            onClick={() => setActiveTab('federated')}
          >
            Federated
          </button>
        </div>
        <div className="nav-right">
          <div style={{ position: 'relative' }}>
            <input 
              type="text" 
              placeholder="Search users..." 
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onFocus={() => searchQuery.length >= 2 && setShowSearchResults(true)}
              className="nav-search"
              style={{ minWidth: '200px' }}
            />
            {/* Search Results Dropdown */}
            {showSearchResults && (
              <div style={{
                position: 'absolute',
                top: '100%',
                left: 0,
                right: 0,
                background: '#1a1a2e',
                border: '1px solid #333',
                borderRadius: '8px',
                marginTop: '4px',
                maxHeight: '300px',
                overflowY: 'auto',
                zIndex: 1000,
                boxShadow: '0 4px 20px rgba(0,0,0,0.5)'
              }}>
                {isSearching ? (
                  <div style={{ padding: '16px', textAlign: 'center', color: '#666' }}>
                    Searching...
                  </div>
                ) : searchResults.length === 0 ? (
                  <div style={{ padding: '16px', textAlign: 'center', color: '#666' }}>
                    No users found
                  </div>
                ) : (
                  searchResults.map(user => (
                    <div 
                      key={user.id}
                      style={{
                        padding: '12px 16px',
                        borderBottom: '1px solid #333',
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '12px',
                        transition: 'background 0.2s'
                      }}
                      onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(0,217,255,0.1)'}
                      onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
                    >
                      <div style={{
                        width: '40px',
                        height: '40px',
                        borderRadius: '50%',
                        background: 'linear-gradient(135deg, #00d9ff, #00ff88)',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontSize: '18px'
                      }}>
                        {user.avatar_url || '👤'}
                      </div>
                      <div style={{ flex: 1 }}>
                        <div style={{ color: '#fff', fontWeight: '600' }}>
                          {user.display_name || user.username}
                        </div>
                        <div style={{ color: '#666', fontSize: '12px' }}>
                          @{user.username}@{user.instance_domain}
                        </div>
                      </div>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          startDMWithUser(user);
                        }}
                        style={{
                          padding: '6px 12px',
                          background: 'rgba(0,217,255,0.2)',
                          border: '1px solid #00d9ff',
                          color: '#00d9ff',
                          borderRadius: '4px',
                          cursor: 'pointer',
                          fontSize: '12px'
                        }}
                      >
                        💬 DM
                      </button>
                    </div>
                  ))
                )}
                {searchResults.length > 0 && (
                  <div 
                    style={{ 
                      padding: '8px 16px', 
                      textAlign: 'center',
                      borderTop: '1px solid #333'
                    }}
                  >
                    <button
                      onClick={() => setShowSearchResults(false)}
                      style={{
                        background: 'none',
                        border: 'none',
                        color: '#666',
                        cursor: 'pointer',
                        fontSize: '12px'
                      }}
                    >
                      Close
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>
          <button 
            className="nav-btn-profile"
            onClick={() => onNavigate('profile')}
          >
            👤 {userData.username || 'User'}
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
          {handleLogout && (
            <button 
              className="nav-btn-profile"
              onClick={handleLogout}
              title="Logout"
              style={{
                marginLeft: '10px',
                padding: '8px 12px',
                background: 'rgba(255, 68, 68, 0.1)',
                border: '1px solid #ff4444',
                color: '#ff4444',
                borderRadius: '6px',
                cursor: 'pointer'
              }}
            >
              🚪
            </button>
          )}
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
                <span className="info-value">{userData.server || userData.instance_domain || 'localhost'}</span>
              </div>
              <div className="info-item">
                <span className="info-label">Your Role</span>
                <span className="info-value" style={{ 
                  color: userData.role === 'admin' ? '#ff4444' : 
                         userData.role === 'moderator' ? '#00d9ff' : '#00ff88'
                }}>
                  {userData.role === 'admin' ? '👑 Admin' : 
                   userData.role === 'moderator' ? '🛡️ Moderator' : '👤 User'}
                </span>
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
          {/* Error display */}
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
                disabled={isPosting}
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
                    className={`composer-btn-post ${!newPostText.trim() || isPosting ? 'disabled' : ''}`}
                    onClick={handlePostCreate}
                    disabled={!newPostText.trim() || isPosting}
                  >
                    {isPosting ? 'Posting...' : 'Post 🚀'}
                  </button>
                </div>
              </div>
            </div>
          </div>

          {/* Feed Divider */}
          <div className="feed-divider" />

          {/* Loading indicator */}
          {isLoading && (
            <div style={{ textAlign: 'center', padding: '20px' }}>
              Loading posts...
            </div>
          )}

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
                  <button 
                    className={`post-action ${post.reposted ? 'active' : ''}`}
                    onClick={() => handleRepost(post.id)}
                    style={post.reposted ? { color: '#00d9ff' } : {}}
                  >
                    <span className="action-icon">🚀</span>
                    <span className="action-count">{post.boosts}</span>
                  </button>
                  <button 
                    className={`post-action ${post.liked ? 'active' : ''}`}
                    onClick={() => handleLike(post.id)}
                    style={post.liked ? { color: '#ff4444' } : {}}
                  >
                    <span className="action-icon">{post.liked ? '❤️' : '🤍'}</span>
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

          {/* Admin Panel - Only for admins/moderators */}
          {(userData.role === 'admin' || userData.role === 'moderator') && (
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
              {userData.role === 'admin' && (
                <button 
                  onClick={() => onNavigate('admin')}
                  style={{
                    width: '100%',
                    padding: '10px',
                    marginTop: '8px',
                    background: 'rgba(255, 68, 68, 0.1)',
                    border: '1px solid #ff4444',
                    color: '#ff4444',
                    borderRadius: '6px',
                    cursor: 'pointer',
                    fontWeight: '600',
                    transition: 'all 0.3s ease'
                  }}
                  className="trend-item"
                >
                  👑 Admin Dashboard
                </button>
              )}
            </div>
          )}

          {/* Request Moderation - Only for regular users */}
          {userData.role === 'user' && (
            <div className="trends-section">
              <h3 className="trends-title">🛡️ Moderation</h3>
              {userData.moderation_requested ? (
                <div style={{
                  padding: '12px',
                  background: 'rgba(255, 170, 0, 0.1)',
                  border: '1px solid #ffaa00',
                  borderRadius: '6px',
                  color: '#ffaa00',
                  fontSize: '14px',
                  textAlign: 'center'
                }}>
                  ⏳ Moderation request pending approval
                </div>
              ) : (
                <button 
                  onClick={async () => {
                    try {
                      await adminApi.requestModeration();
                      if (updateUserData) {
                        updateUserData({ ...userData, moderation_requested: true });
                      }
                      alert('Moderation request submitted! An admin will review it.');
                    } catch (err) {
                      alert('Failed to submit request: ' + err.message);
                    }
                  }}
                  style={{
                    width: '100%',
                    padding: '10px',
                    background: 'rgba(0, 255, 136, 0.1)',
                    border: '1px solid #00ff88',
                    color: '#00ff88',
                    borderRadius: '6px',
                    cursor: 'pointer',
                    fontWeight: '600',
                    transition: 'all 0.3s ease'
                  }}
                  className="trend-item"
                >
                  🙋 Request Moderation Access
                </button>
              )}
              <p style={{ 
                fontSize: '12px', 
                color: '#666', 
                marginTop: '8px',
                lineHeight: '1.4'
              }}>
                Want to help moderate this server? Request access and an admin will review your application.
              </p>
            </div>
          )}

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
