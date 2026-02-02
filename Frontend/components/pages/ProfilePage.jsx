'use client';

import { useState, useEffect } from 'react';
import { useTheme } from '@/components/ui/theme-provider';
import '../styles/ProfilePage.css';
import { followApi, userApi, postApi } from '@/lib/api';

export default function ProfilePage({ onNavigate, userData, viewingUserId = null }) {
  const { theme, toggleTheme } = useTheme();
  const isDarkMode = theme === 'dark';
  const [activeTab, setActiveTab] = useState('posts');
  const [isFollowing, setIsFollowing] = useState(false);
  const [isFollowLoading, setIsFollowLoading] = useState(false);
  const [profileData, setProfileData] = useState(null);
  const [isLoadingProfile, setIsLoadingProfile] = useState(false);
  const [stats, setStats] = useState({ followers: 0, following: 0, posts: 0 });
  const [userPosts, setUserPosts] = useState([]);

  // Fetch user profile and stats
  useEffect(() => {
    const fetchProfileAndStats = async () => {
      const targetId = viewingUserId || userData?.id;
      if (!targetId) {
        setProfileData(userData);
        return;
      }

      setIsLoadingProfile(true);
      try {
        let profile;
        if (viewingUserId) {
          // Viewing another user's profile
          profile = await userApi.getUserProfile(viewingUserId);
          setProfileData({
            username: profile.username,
            server: profile.instance_domain || 'localhost:8000',
            displayName: profile.display_name || profile.username,
            avatar: profile.avatar_url || '👤',
            bio: profile.bio || 'No bio yet',
            email: profile.email,
            did: profile.did,
          });

          // Check if current user is following this user
          if (userData?.id) {
            try {
              const following = await followApi.getFollowing(userData.id);
              const isCurrentlyFollowing = (following || []).some(u => u.id === viewingUserId);
              setIsFollowing(isCurrentlyFollowing);
            } catch (err) {
              console.error('Failed to check follow status:', err);
            }
          }
        } else {
          // Viewing own profile
          setProfileData(userData);
        }

        // Fetch follow stats
        try {
          const followStats = await followApi.getFollowStats(targetId);
          setStats(prev => ({
            ...prev,
            followers: followStats.followers || 0,
            following: followStats.following || 0,
          }));
        } catch (err) {
          console.error('Failed to fetch follow stats:', err);
        }

        // Fetch post count (get user's posts)
        try {
          const targetDid = viewingUserId ? profile?.did : userData?.did;
          if (targetDid) {
            const posts = await postApi.getUserPosts(targetDid);
            setStats(prev => ({
              ...prev,
              posts: (posts || []).length,
            }));
            setUserPosts(posts || []);
          }
        } catch (err) {
          console.error('Failed to fetch user posts:', err);
        }

      } catch (err) {
        console.error('Failed to load profile:', err);
        setProfileData(userData);
      } finally {
        setIsLoadingProfile(false);
      }
    };

    fetchProfileAndStats();
  }, [viewingUserId, userData]);

  // Handle follow/unfollow
  const handleFollowToggle = async () => {
    if (!viewingUserId) return; // Can't follow yourself

    setIsFollowLoading(true);
    try {
      if (isFollowing) {
        await followApi.unfollowUser(viewingUserId);
        setIsFollowing(false);
        setStats(prev => ({ ...prev, followers: Math.max(0, prev.followers - 1) }));
      } else {
        await followApi.followUser(viewingUserId);
        setIsFollowing(true);
        setStats(prev => ({ ...prev, followers: prev.followers + 1 }));
      }
    } catch (err) {
      console.error('Follow operation failed:', err);
      alert(`Failed to ${isFollowing ? 'unfollow' : 'follow'} user: ${err.message}`);
    } finally {
      setIsFollowLoading(false);
    }
  };

  // Use fetched profile data
  const displayData = profileData || userData;
  const profile = {
    username: displayData?.username || 'unknown',
    server: displayData?.server || 'localhost:8000',
    displayName: displayData?.displayName || displayData?.display_name || displayData?.username || 'Unknown User',
    did: displayData?.did || 'did:key:...',
    reputation: 'Trusted',
    reputationColor: '#00d9ff',
    avatar: displayData?.avatar || '👤',
    bio: displayData?.bio || 'No bio yet',
    email: displayData?.email,
    isLocal: true,
    followers: stats.followers,
    following: stats.following,
    posts: stats.posts,
  };

  // Format post timestamp
  const formatTimestamp = (timestamp) => {
    if (!timestamp) return 'recently';
    const date = new Date(timestamp);
    const now = new Date();
    const diff = Math.floor((now - date) / 1000);

    if (diff < 60) return 'just now';
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    if (diff < 604800) return `${Math.floor(diff / 86400)}d ago`;
    return date.toLocaleDateString();
  };

  if (isLoadingProfile) {
    return (
      <div className="profile-container" style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        <div style={{ textAlign: 'center', color: '#00d9ff' }}>
          <div style={{ fontSize: '48px', marginBottom: '16px' }}>⏳</div>
          <div>Loading profile...</div>
        </div>
      </div>
    );
  }

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
          {!viewingUserId && (
            <button onClick={() => onNavigate('thread')} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>💬 Threads</button>
          )}
          {viewingUserId ? (
            <button onClick={() => onNavigate('dm', { selectedUser: { id: viewingUserId, username: displayData.username, display_name: displayData.displayName } })} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>💬 Message User</button>
          ) : (
            <button onClick={() => onNavigate('dm')} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>📨 Messages</button>
          )}
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
                onClick={handleFollowToggle}
                disabled={isFollowLoading || !viewingUserId}
                style={{
                  opacity: isFollowLoading ? 0.6 : 1,
                  cursor: isFollowLoading || !viewingUserId ? 'not-allowed' : 'pointer'
                }}
              >
                {isFollowLoading ? '...' : isFollowing ? '✓ Following' : 'Follow'}
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
            {userPosts.length === 0 ? (
              <div style={{ textAlign: 'center', color: '#666', padding: '32px' }}>
                No posts yet
              </div>
            ) : (
              userPosts.map((post) => (
                <div
                  key={post.id}
                  className="post-card-profile"
                  onClick={() => onNavigate('thread')}
                  style={{ cursor: 'pointer' }}
                >
                  <div className="post-badges">
                    {post.is_remote ? (
                      <span className="federated-badge">🌐 Remote</span>
                    ) : (
                      <span className="local-badge">Local</span>
                    )}
                    {post.visibility === 'followers' && (
                      <span className="followers-badge">👥 Followers Only</span>
                    )}
                    {post.visibility === 'circle' && (
                      <span className="followers-badge">🔒 Circle</span>
                    )}
                  </div>
                  <div className="post-content">{post.content}</div>
                  <div className="post-timestamp">{formatTimestamp(post.created_at)}</div>
                </div>
              ))
            )}
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
