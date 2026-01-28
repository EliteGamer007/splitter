'use client';

import { useState } from 'react';
import '../styles/ProfilePage.css';

export default function ProfilePage({ onNavigate, userData, isDarkMode, toggleTheme, viewingUserId = null }) {
  const [activeTab, setActiveTab] = useState('posts');
  const [isFollowing, setIsFollowing] = useState(false);

  // Use passed userData or mock profile data
  const profile = {
    username: userData.username,
    server: userData.server,
    displayName: userData.displayName,
    did: 'did:key:z6Mkg5r9Z4x2bK7nQ9pL2m8vN3tC5wJ6...',
    reputation: 'Trusted',
    reputationColor: '#00d9ff',
    avatar: userData.avatar,
    bio: userData.bio,
    email: userData.email,
    isLocal: true,
    followers: userData.followers,
    following: userData.following,
    posts: userData.postsCount,
  };

  const userPosts = [
    {
      id: 1,
      content: 'Just deployed a new federated instance! 🚀',
      timestamp: '2 hours ago',
      isLocal: true,
      isFollowersOnly: false,
    },
    {
      id: 2,
      content: 'Understanding DIDs has been a game-changer for decentralized auth.',
      timestamp: '1 day ago',
      isLocal: true,
      isFollowersOnly: false,
    },
    {
      id: 3,
      content: 'Thoughts on ActivityPub compliance and federation standards.',
      timestamp: '3 days ago',
      isLocal: true,
      isFollowersOnly: false,
    },
  ];

  return (
    <div className="profile-container">
      {/* Top Navigation Bar */}
      <div className="profile-navbar">
        <div className="navbar-left">
          <button
            className="nav-button back-button"
            onClick={() => onNavigate('home')}
          >
            ← Back
          </button>
          <h1 className="navbar-title">Profile</h1>
        </div>
        <div className="navbar-center">
          <button className="nav-badge">Local</button>
          <button className="nav-badge">Federated</button>
        </div>
        <div style={{ marginLeft: 'auto', display: 'flex', gap: '10px' }}>
          <button onClick={() => onNavigate('thread')} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>💬 Threads</button>
          <button onClick={() => onNavigate('dm')} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>📨 Messages</button>
          <button onClick={() => onNavigate('security')} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>🔐 Security</button>
          <button onClick={toggleTheme} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>{isDarkMode ? '🌙' : '☀️'}</button>
        </div>
      </div>

      <div className="profile-content">
        {/* Profile Header Card */}
        <div className="profile-header-card">
          <div className="profile-header-top">
            <div className="profile-avatar-section">
              <div className="profile-avatar">{profile.avatar}</div>
              <div className="profile-info">
                <div className="profile-username">
                  @{profile.username}@{profile.server}
                  {!profile.isLocal && <span className="federated-badge">🌐</span>}
                </div>
                <div className="profile-reputation">
                  <span
                    className="reputation-dot"
                    style={{ backgroundColor: profile.reputationColor }}
                  ></span>
                  <span className="reputation-text">
                    {profile.reputation}
                    <span className="disabled-tooltip">
                      ⓘ Reputation scoring enabled in Sprint 3
                    </span>
                  </span>
                </div>
              </div>
            </div>

            <div className="profile-actions">
              <button
                className={`follow-button ${isFollowing ? 'following' : ''}`}
                onClick={() => setIsFollowing(!isFollowing)}
              >
                {isFollowing ? '✓ Following' : 'Follow'}
              </button>
              <button className="message-button" title="DMs available on /dm page">
                Message 🔒
              </button>
            </div>
          </div>

          {/* DID Display */}
          <div className="profile-did-section">
            <div className="did-label">Decentralized Identifier</div>
            <div className="did-value">{profile.did}</div>
          </div>

          {/* Bio */}
          <div className="profile-bio">{profile.bio}</div>

          {/* Stats Bar */}
          <div className="profile-stats">
            <div className="stat-item">
              <div className="stat-number">{profile.posts}</div>
              <div className="stat-label">Posts</div>
            </div>
            <div className="stat-item">
              <div className="stat-number">{profile.followers}</div>
              <div className="stat-label">Followers</div>
            </div>
            <div className="stat-item">
              <div className="stat-number">{profile.following}</div>
              <div className="stat-label">Following</div>
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="profile-tabs">
          <button
            className={`tab-button ${activeTab === 'posts' ? 'active' : ''}`}
            onClick={() => setActiveTab('posts')}
          >
            Posts
          </button>
          <button
            className={`tab-button ${activeTab === 'followers' ? 'active' : ''}`}
            onClick={() => setActiveTab('followers')}
          >
            Followers
          </button>
          <button
            className={`tab-button ${activeTab === 'following' ? 'active' : ''}`}
            onClick={() => setActiveTab('following')}
          >
            Following
          </button>
        </div>

        {/* Posts Tab */}
        {activeTab === 'posts' && (
          <div className="profile-posts-list">
            {userPosts.map((post) => (
              <div
                key={post.id}
                className="post-card-profile"
                onClick={() => onNavigate('thread')}
                style={{ cursor: 'pointer' }}
              >
                <div className="post-badges">
                  {post.isLocal ? (
                    <span className="local-badge">🏠 Local</span>
                  ) : (
                    <span className="federated-badge">🌐 Remote</span>
                  )}
                  {post.isFollowersOnly && (
                    <span className="followers-badge">👥 Followers Only</span>
                  )}
                </div>
                <div className="post-content">{post.content}</div>
                <div className="post-timestamp">{post.timestamp}</div>
              </div>
            ))}
          </div>
        )}

        {/* Followers Tab */}
        {activeTab === 'followers' && (
          <div className="profile-list">
            <div className="list-item">
              <div className="follower-avatar">👨</div>
              <div className="follower-info">
                <div className="follower-name">@bob@federated.social</div>
                <div className="follower-status">Trusted</div>
              </div>
              <button className="unfollow-button">Unfollow</button>
            </div>
            <div className="list-item">
              <div className="follower-avatar">👩</div>
              <div className="follower-info">
                <div className="follower-name">@carol@social.example.net</div>
                <div className="follower-status">Local</div>
              </div>
              <button className="unfollow-button">Unfollow</button>
            </div>
          </div>
        )}

        {/* Following Tab */}
        {activeTab === 'following' && (
          <div className="profile-list">
            <div className="list-item">
              <div className="follower-avatar">🧑</div>
              <div className="follower-info">
                <div className="follower-name">@dave@crypto.social</div>
                <div className="follower-status">Remote</div>
              </div>
              <button className="unfollow-button">Following</button>
            </div>
            <div className="list-item">
              <div className="follower-avatar">👩‍🔬</div>
              <div className="follower-info">
                <div className="follower-name">@eve@research.net</div>
                <div className="follower-status">Trusted</div>
              </div>
              <button className="unfollow-button">Following</button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
