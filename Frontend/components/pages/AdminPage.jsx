'use client';

import { useState, useEffect } from 'react';
import { useTheme } from '@/components/ui/theme-provider';
import '../styles/HomePage.css';
import { adminApi, postApi, searchApi } from '@/lib/api';

export default function AdminPage({ onNavigate, userData, handleLogout }) {
  /* ================= THEME (from your branch) ================= */
  const { theme, toggleTheme } = useTheme();
  const isDarkMode = theme === 'dark';

  /* ================= STATE (from main branch – unchanged) ================= */
  const [activeTab, setActiveTab] = useState('home');
  const [posts, setPosts] = useState([]);
  const [users, setUsers] = useState([]);
  const [moderationRequests, setModerationRequests] = useState([]);
  const [suspendedUsers, setSuspendedUsers] = useState([]);
  const [adminActions, setAdminActions] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [totalUsers, setTotalUsers] = useState(0);
  const [page, setPage] = useState(0);
  const [actionLoading, setActionLoading] = useState(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [showSearchResults, setShowSearchResults] = useState(false);
  const [searchResults, setSearchResults] = useState([]);
  const [isSearching, setIsSearching] = useState(false);

  const [showBanModal, setShowBanModal] = useState(false);
  const [banTarget, setBanTarget] = useState(null);
  const [banReason, setBanReason] = useState('');

  /* ================= ACCESS CHECK ================= */
  if (userData?.role !== 'admin') {
    return (
      <div style={{
        minHeight: '100vh',
        background: 'var(--bg-primary)',
        color: 'var(--text-primary)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center'
      }}>
        <h2>⛔ Access Denied</h2>
      </div>
    );
  }

  /* ================= EFFECTS ================= */
  useEffect(() => {
    loadTabData();
  }, [activeTab, page]);

  /* ================= DATA LOADERS ================= */

  const loadTabData = async () => {
    setIsLoading(true);
    try {
      if (activeTab === 'home') fetchPosts();
      if (activeTab === 'requests') fetchModerationRequests();
      if (activeTab === 'bans') {
        fetchSuspendedUsers();
        fetchAdminActions();
      }
      if (activeTab === 'users') fetchAllUsers();
    } finally {
      setIsLoading(false);
    }
  };

  const fetchPosts = async () => {
    const feedPosts = await postApi.getPublicFeed(20, 0, false);
    setPosts(feedPosts || []);
  };

  const fetchModerationRequests = async () => {
    const result = await adminApi.getModerationRequests();
    setModerationRequests(result.requests || []);
  };

  const fetchSuspendedUsers = async () => {
    const result = await adminApi.getSuspendedUsers(100, 0);
    setSuspendedUsers(result.users || []);
  };

  const fetchAdminActions = async () => {
    const result = await adminApi.getAdminActions();
    setAdminActions(result.actions || []);
  };

  const fetchAllUsers = async () => {
    const result = await adminApi.getAllUsers(50, page * 50);
    setUsers(result.users || []);
    setTotalUsers(result.total || 0);
  };

  /* ================= HANDLERS ================= */

  const handleDeletePost = async (postId) => {
    await postApi.deletePost(postId);
    setPosts(posts.filter(p => p.id !== postId));
  };

  const formatTimestamp = (iso) => {
    if (!iso) return '';
    const diff = Date.now() - new Date(iso);
    const mins = Math.floor(diff / 60000);
    if (mins < 60) return `${mins}m ago`;
    return new Date(iso).toLocaleDateString();
  };

  /* ================= UI (UNCHANGED FROM MAIN) ================= */

  return (
    <div className="home-container">

      {/* ================= NAVBAR ================= */}
      <nav className="home-nav">

        <div className="nav-left">
          <img src="/logo.png" className="nav-logo-img" />
        </div>

        <div className="nav-center">
          <button 
            className={`nav-item ${activeTab === 'home' ? 'active' : ''}`}
            onClick={() => setActiveTab('home')}
          >
            Feed
          </button>
          <button 
            className={`nav-item ${activeTab === 'requests' ? 'active' : ''}`}
            onClick={() => setActiveTab('requests')}
            style={{ position: 'relative' }}
          >
            Requests
            {moderationRequests.length > 0 && (
              <span style={{
                position: 'absolute',
                top: '-5px',
                right: '-5px',
                background: '#ff4444',
                color: '#fff',
                borderRadius: '50%',
                width: '20px',
                height: '20px',
                fontSize: '11px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center'
              }}>
                {moderationRequests.length}
              </span>
            )}
          </button>
          <button 
            className={`nav-item ${activeTab === 'bans' ? 'active' : ''}`}
            onClick={() => setActiveTab('bans')}
          >
            Bans
          </button>
          <button 
            className={`nav-item ${activeTab === 'users' ? 'active' : ''}`}
            onClick={() => setActiveTab('users')}
          >
            Users
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
              style={{ 
                minWidth: showSearchResults ? '400px' : '300px',
                width: showSearchResults ? '450px' : '300px',
                transition: 'all 0.3s ease'
              }}
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
                        display: 'flex',
                        alignItems: 'center',
                        gap: '12px'
                      }}
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
                          @{user.username} · {user.role}
                          {user.is_suspended && <span style={{ color: '#ff4444', marginLeft: '8px' }}>Suspended</span>}
                        </div>
                      </div>
                      <div style={{ display: 'flex', gap: '8px' }}>
                        {!user.is_suspended ? (
                          <button
                            onClick={() => {
                              setShowSearchResults(false);
                              openBanModal(user);
                            }}
                            style={{
                              padding: '6px 12px',
                              background: 'rgba(255,68,68,0.2)',
                              border: '1px solid #ff4444',
                              color: '#ff4444',
                              borderRadius: '4px',
                              cursor: 'pointer',
                              fontSize: '12px'
                            }}
                          >
                            Ban
                          </button>
                        ) : (
                          <button
                            onClick={() => {
                              setShowSearchResults(false);
                              handleUnsuspendUser(user.id, user.username);
                            }}
                            style={{
                              padding: '6px 12px',
                              background: 'rgba(0,255,136,0.2)',
                              border: '1px solid #00ff88',
                              color: '#00ff88',
                              borderRadius: '4px',
                              cursor: 'pointer',
                              fontSize: '12px'
                            }}
                          >
                            Unban
                          </button>
                        )}
                      </div>
                    </div>
                  ))
                )}
              </div>
            )}
          </div>
          <button 
            onClick={toggleTheme}
            className="nav-btn"
            title="Toggle theme"
          >
            {isDarkMode ? '🌙' : '☀️'}
          </button>

          <button onClick={handleLogout}>Logout</button>
        </div>
      </nav>

      {/* Main Layout */}
      <div className="admin-layout">
        {/* ================= MAIN CONTENT ================= */}
        <main className="home-feed">
        {/* TAB: Feed/Home */}
        {activeTab === 'home' && (
          <>
            <div className="feed-header">
              <h2>Feed Monitoring</h2>
              <p style={{ color: '#666', fontSize: '14px' }}>
                Recent public posts from all users
              </p>
            </div>

            {isLoading ? (
              <div style={{ textAlign: 'center', padding: '40px', color: '#666' }}>Loading posts...</div>
            ) : posts.length === 0 ? (
              <div style={{ textAlign: 'center', padding: '60px', background: 'rgba(255,255,255,0.02)', border: '1px solid #333', borderRadius: '12px' }}>
                <p style={{ color: '#666' }}>No posts yet</p>
              </div>
            ) : (
              posts.map(post => (
                <article key={post.id} style={{
                  background: 'rgba(255,255,255,0.02)',
                  border: '1px solid #333',
                  borderRadius: '12px',
                  padding: '20px',
                  marginBottom: '16px'
                }}>
                  <div style={{ display: 'flex', gap: '12px', marginBottom: '12px' }}>
                    <div style={{
                      width: '48px',
                      height: '48px',
                      borderRadius: '50%',
                      background: 'linear-gradient(135deg, #00d9ff, #00ff88)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      flexShrink: 0
                    }}>
                      {post.author_avatar || '👤'}
                    </div>
                    <div style={{ flex: 1 }}>
                      <div style={{ fontWeight: '600', marginBottom: '4px' }}>
                        {post.author_display_name || post.author_username}
                      </div>
                      <div style={{ color: '#666', fontSize: '13px' }}>
                        @{post.author_username} · {formatDate(post.created_at)}
                      </div>
                    </div>
                    <button
                      onClick={() => handleDeletePost(post.id)}
                      style={{
                        padding: '6px 12px',
                        background: 'rgba(255,68,68,0.2)',
                        border: '1px solid #ff4444',
                        color: '#ff4444',
                        borderRadius: '6px',
                        cursor: 'pointer',
                        fontSize: '12px'
                      }}
                    >
                      Delete
                    </button>
                  </div>
                  <p style={{ margin: '0 0 12px 60px', lineHeight: '1.5' }}>
                    {post.content}
                  </p>
                </article>
              ))
            )}
          </>
        )}

          {/* TAB: Moderation Requests */}
          {activeTab === 'requests' && (
            <>
              <div className="feed-header">
                <h2>Moderation Requests</h2>
                <p style={{ color: '#666', fontSize: '14px' }}>
                  Users requesting moderator privileges ({moderationRequests.length} pending)
                </p>
              </div>

              {isLoading ? (
                <div style={{ textAlign: 'center', padding: '40px', color: '#666' }}>
                  Loading requests...
                </div>
              ) : moderationRequests.length === 0 ? (
                <div style={{
                  textAlign: 'center',
                  padding: '60px',
                  background: 'rgba(255,255,255,0.02)',
                  border: '1px solid #333',
                  borderRadius: '12px'
                }}>
                  <h3 style={{ marginBottom: '8px' }}>No pending requests</h3>
                  <p style={{ color: '#666' }}>All moderation requests have been processed.</p>
                </div>
              ) : (
                <div style={{ display: 'grid', gap: '16px' }}>
                  {moderationRequests.map(user => (
                    <div
                      key={user.id}
                      style={{
                        background: 'rgba(255,255,255,0.02)',
                        border: '1px solid #333',
                        borderRadius: '12px',
                        padding: '20px',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between'
                      }}
                    >
                      <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                        <div style={{
                          width: '60px',
                          height: '60px',
                          borderRadius: '50%',
                          background: 'linear-gradient(135deg, #00d9ff, #00ff88)',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          fontSize: '28px'
                        }}>
                          {user.avatar_url || '👤'}
                        </div>
                        <div>
                          <div style={{ fontWeight: '600', fontSize: '18px' }}>
                            {user.display_name || user.username}
                          </div>
                          <div style={{ color: '#00d9ff' }}>@{user.username}</div>
                          <div style={{ color: '#666', fontSize: '12px', marginTop: '4px' }}>
                            Member since {formatDate(user.created_at)}
                          </div>
                          {user.bio && (
                            <div style={{ color: '#888', fontSize: '13px', marginTop: '8px', maxWidth: '400px' }}>
                              "{user.bio}"
                            </div>
                          )}
                        </div>
                      </div>
                      <div style={{ display: 'flex', gap: '12px' }}>
                        <button
                          onClick={() => handleApproveModeration(user.id)}
                          disabled={actionLoading === user.id}
                          style={{
                            padding: '12px 24px',
                            background: 'rgba(0,255,136,0.2)',
                            border: '1px solid #00ff88',
                            color: '#00ff88',
                            borderRadius: '8px',
                            cursor: 'pointer',
                            fontWeight: '600',
                            fontSize: '14px'
                          }}
                        >
                          {actionLoading === user.id ? '...' : '✓ Approve'}
                        </button>
                        <button
                          onClick={() => handleRejectModeration(user.id)}
                          disabled={actionLoading === user.id}
                          style={{
                            padding: '12px 24px',
                            background: 'rgba(255,68,68,0.2)',
                            border: '1px solid #ff4444',
                            color: '#ff4444',
                            borderRadius: '8px',
                            cursor: 'pointer',
                            fontWeight: '600',
                            fontSize: '14px'
                          }}
                        >
                          {actionLoading === user.id ? '...' : '✕ Reject'}
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </>
          )}

          {/* TAB: Bans */}
          {activeTab === 'bans' && (
            <>
              <div className="feed-header">
                <h2>Ban Management</h2>
                <p style={{ color: '#666', fontSize: '14px' }}>
                  View suspended users and admin action history
                </p>
              </div>

              {/* Currently Suspended Users */}
              <div style={{ marginBottom: '32px' }}>
                <h3 style={{ marginBottom: '16px', color: '#ff4444' }}>
                  🔴 Currently Suspended ({suspendedUsers.length})
                </h3>
                {isLoading ? (
                  <div style={{ textAlign: 'center', padding: '20px', color: '#666' }}>Loading...</div>
                ) : suspendedUsers.length === 0 ? (
                  <div style={{
                    padding: '30px',
                    background: 'rgba(0,255,136,0.05)',
                    border: '1px solid #333',
                    borderRadius: '12px',
                    textAlign: 'center'
                  }}>
                    <span style={{ fontSize: '32px' }}>✅</span>
                    <p style={{ color: '#00ff88', marginTop: '8px' }}>No users are currently suspended</p>
                  </div>
                ) : (
                  <div style={{ display: 'grid', gap: '12px' }}>
                    {suspendedUsers.map(user => (
                      <div
                        key={user.id}
                        style={{
                          background: 'rgba(255,68,68,0.05)',
                          border: '1px solid rgba(255,68,68,0.3)',
                          borderRadius: '12px',
                          padding: '16px',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'space-between'
                        }}
                      >
                        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                          <div style={{
                            width: '50px',
                            height: '50px',
                            borderRadius: '50%',
                            background: 'rgba(255,68,68,0.2)',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            fontSize: '24px',
                            filter: 'grayscale(50%)'
                          }}>
                            {user.avatar_url || '👤'}
                          </div>
                          <div>
                            <div style={{ fontWeight: '600', color: '#ff4444' }}>
                              {user.display_name || user.username}
                            </div>
                            <div style={{ color: '#888', fontSize: '13px' }}>
                              @{user.username} · Suspended
                            </div>
                          </div>
                        </div>
                        <button
                          onClick={() => handleUnsuspendUser(user.id, user.username)}
                          disabled={actionLoading === user.id}
                          style={{
                            padding: '10px 20px',
                            background: 'rgba(0,255,136,0.2)',
                            border: '1px solid #00ff88',
                            color: '#00ff88',
                            borderRadius: '6px',
                            cursor: 'pointer',
                            fontWeight: '600'
                          }}
                        >
                          {actionLoading === user.id ? '...' : '✓ Unsuspend'}
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Admin Action History */}
              <div>
                <h3 style={{ marginBottom: '16px', color: '#00d9ff' }}>
                  📜 Recent Admin Actions
                </h3>
                {adminActions.length === 0 ? (
                  <div style={{
                    padding: '30px',
                    background: 'rgba(255,255,255,0.02)',
                    border: '1px solid #333',
                    borderRadius: '12px',
                    textAlign: 'center'
                  }}>
                    <p style={{ color: '#666' }}>No admin actions recorded yet.</p>
                  </div>
                ) : (
                  <div style={{
                    background: 'rgba(255,255,255,0.02)',
                    border: '1px solid #333',
                    borderRadius: '12px',
                    overflow: 'hidden'
                  }}>
                    <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                      <thead>
                        <tr style={{ background: 'rgba(0,0,0,0.3)' }}>
                          <th style={{ padding: '12px 16px', textAlign: 'left', color: '#888' }}>Action</th>
                          <th style={{ padding: '12px 16px', textAlign: 'left', color: '#888' }}>Target</th>
                          <th style={{ padding: '12px 16px', textAlign: 'left', color: '#888' }}>Reason</th>
                          <th style={{ padding: '12px 16px', textAlign: 'left', color: '#888' }}>Date</th>
                        </tr>
                      </thead>
                      <tbody>
                        {adminActions.map((action, idx) => (
                          <tr key={idx} style={{ borderTop: '1px solid #333' }}>
                            <td style={{ padding: '12px 16px' }}>
                              <span style={{
                                padding: '4px 8px',
                                borderRadius: '4px',
                                fontSize: '12px',
                                background: action.action_type === 'suspend' ? 'rgba(255,68,68,0.2)' : 'rgba(0,255,136,0.2)',
                                color: action.action_type === 'suspend' ? '#ff4444' : '#00ff88'
                              }}>
                                {action.action_type}
                              </span>
                            </td>
                            <td style={{ padding: '12px 16px', color: '#fff' }}>{action.target || 'N/A'}</td>
                            <td style={{ padding: '12px 16px', color: '#888' }}>{action.reason || '—'}</td>
                            <td style={{ padding: '12px 16px', color: '#666' }}>{formatDate(action.created_at)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </>
          )}

          {/* TAB: All Users */}
          {activeTab === 'users' && (
            <>
              <div className="feed-header">
                <h2>All Users</h2>
                <p style={{ color: '#666', fontSize: '14px' }}>
                  {totalUsers} total users on this instance
                </p>
              </div>

              {isLoading ? (
                <div style={{ textAlign: 'center', padding: '40px', color: '#666' }}>Loading users...</div>
              ) : (
                <>
                  <div style={{
                    background: 'rgba(255,255,255,0.02)',
                    border: '1px solid #333',
                    borderRadius: '12px',
                    overflow: 'hidden'
                  }}>
                    <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                      <thead>
                        <tr style={{ background: 'rgba(0,0,0,0.3)' }}>
                          <th style={{ padding: '16px', textAlign: 'left', color: '#888' }}>User</th>
                          <th style={{ padding: '16px', textAlign: 'left', color: '#888' }}>Role</th>
                          <th style={{ padding: '16px', textAlign: 'left', color: '#888' }}>Status</th>
                          <th style={{ padding: '16px', textAlign: 'left', color: '#888' }}>Joined</th>
                          <th style={{ padding: '16px', textAlign: 'right', color: '#888' }}>Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {users.map(user => (
                          <tr key={user.id} style={{ borderTop: '1px solid #333' }}>
                            <td style={{ padding: '16px' }}>
                              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                                <div style={{
                                  width: '40px',
                                  height: '40px',
                                  borderRadius: '50%',
                                  background: user.is_suspended 
                                    ? 'rgba(255,68,68,0.2)' 
                                    : 'linear-gradient(135deg, #00d9ff, #00ff88)',
                                  display: 'flex',
                                  alignItems: 'center',
                                  justifyContent: 'center',
                                  filter: user.is_suspended ? 'grayscale(50%)' : 'none'
                                }}>
                                  {user.avatar_url || '👤'}
                                </div>
                                <div>
                                  <div style={{ fontWeight: '600' }}>{user.display_name || user.username}</div>
                                  <div style={{ color: '#666', fontSize: '12px' }}>@{user.username}</div>
                                </div>
                              </div>
                            </td>
                            <td style={{ padding: '16px' }}>
                              <span style={{
                                padding: '4px 8px',
                                borderRadius: '4px',
                                fontSize: '12px',
                                fontWeight: '600',
                                background: user.role === 'admin' ? 'rgba(255,68,68,0.2)' :
                                           user.role === 'moderator' ? 'rgba(0,217,255,0.2)' : 'rgba(0,255,136,0.2)',
                                color: user.role === 'admin' ? '#ff4444' :
                                       user.role === 'moderator' ? '#00d9ff' : '#00ff88'
                              }}>
                                {user.role === 'admin' ? '👑' : user.role === 'moderator' ? '🛡️' : '👤'} {user.role}
                              </span>
                            </td>
                            <td style={{ padding: '16px' }}>
                              {user.is_suspended ? (
                                <span style={{ color: '#ff4444' }}>Suspended</span>
                              ) : (
                                <span style={{ color: '#00ff88' }}>✓ Active</span>
                              )}
                            </td>
                            <td style={{ padding: '16px', color: '#888' }}>{formatDate(user.created_at)}</td>
                            <td style={{ padding: '16px', textAlign: 'right' }}>
                              {user.role !== 'admin' && (
                                <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
                                  {user.is_suspended ? (
                                    <button
                                      onClick={() => handleUnsuspendUser(user.id, user.username)}
                                      disabled={actionLoading === user.id}
                                      style={{
                                        padding: '6px 12px',
                                        background: 'rgba(0,255,136,0.2)',
                                        border: '1px solid #00ff88',
                                        color: '#00ff88',
                                        borderRadius: '4px',
                                        cursor: 'pointer',
                                        fontSize: '12px'
                                      }}
                                    >
                                      Unban
                                    </button>
                                  ) : (
                                    <button
                                      onClick={() => openBanModal(user)}
                                      disabled={actionLoading === user.id}
                                      style={{
                                        padding: '6px 12px',
                                        background: 'rgba(255,68,68,0.2)',
                                        border: '1px solid #ff4444',
                                        color: '#ff4444',
                                        borderRadius: '4px',
                                        cursor: 'pointer',
                                        fontSize: '12px'
                                      }}
                                    >
                                      Ban
                                    </button>
                                  )}
                                  <select
                                    value={user.role}
                                    onChange={(e) => handleUpdateRole(user.id, e.target.value)}
                                    disabled={actionLoading === user.id}
                                    style={{
                                      padding: '6px 8px',
                                      background: '#1a1a2e',
                                      border: '1px solid #333',
                                      color: '#fff',
                                      borderRadius: '4px',
                                      cursor: 'pointer',
                                      fontSize: '12px'
                                    }}
                                  >
                                    <option value="user">User</option>
                                    <option value="moderator">Moderator</option>
                                    <option value="admin">Admin</option>
                                  </select>
                                </div>
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>

                  {/* Pagination */}
                  {totalUsers > 50 && (
                    <div style={{ display: 'flex', justifyContent: 'center', gap: '8px', marginTop: '24px' }}>
                      <button
                        onClick={() => setPage(p => Math.max(0, p - 1))}
                        disabled={page === 0}
                        style={{
                          padding: '8px 16px',
                          background: 'rgba(0,217,255,0.1)',
                          border: '1px solid #00d9ff',
                          color: '#00d9ff',
                          borderRadius: '6px',
                          cursor: page === 0 ? 'not-allowed' : 'pointer',
                          opacity: page === 0 ? 0.5 : 1
                        }}
                      >
                        ← Previous
                      </button>
                      <span style={{ padding: '8px 16px', color: '#666' }}>
                        Page {page + 1} of {Math.ceil(totalUsers / 50)}
                      </span>
                      <button
                        onClick={() => setPage(p => p + 1)}
                        disabled={(page + 1) * 50 >= totalUsers}
                        style={{
                          padding: '8px 16px',
                          background: 'rgba(0,217,255,0.1)',
                          border: '1px solid #00d9ff',
                          color: '#00d9ff',
                          borderRadius: '6px',
                          cursor: (page + 1) * 50 >= totalUsers ? 'not-allowed' : 'pointer',
                          opacity: (page + 1) * 50 >= totalUsers ? 0.5 : 1
                        }}
                      >
                        Next →
                      </button>
                    </div>
                  )}
                </>
              )}
            </>
          )}
        </main>

        {/* Right Sidebar - Admin Quick Stats */}
        <aside className="right-sidebar">
          <div className="sidebar-card">
            <h3>📊 Instance Stats</h3>
            <div style={{ display: 'grid', gap: '12px', marginTop: '16px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span style={{ color: '#888' }}>Total Users</span>
                <span style={{ color: '#00d9ff', fontWeight: '600' }}>{totalUsers}</span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span style={{ color: '#888' }}>Pending Requests</span>
                <span style={{ color: '#ff006e', fontWeight: '600' }}>{moderationRequests.length}</span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span style={{ color: '#888' }}>Suspended Users</span>
                <span style={{ color: '#ff4444', fontWeight: '600' }}>{suspendedUsers.length}</span>
              </div>
            </div>
          </div>

          <div className="sidebar-card">
            <h3>🔧 Quick Actions</h3>
            <div style={{ display: 'grid', gap: '8px', marginTop: '16px' }}>
              <button
                onClick={() => onNavigate('moderation')}
                style={{
                  width: '100%',
                  padding: '12px',
                  background: 'rgba(0,217,255,0.1)',
                  border: '1px solid #00d9ff',
                  color: '#00d9ff',
                  borderRadius: '8px',
                  cursor: 'pointer',
                  fontWeight: '500'
                }}
              >
                Content Moderation
              </button>
              <button
                onClick={() => onNavigate('federation')}
                style={{
                  width: '100%',
                  padding: '12px',
                  background: 'rgba(255,0,110,0.1)',
                  border: '1px solid #ff006e',
                  color: '#ff006e',
                  borderRadius: '8px',
                  cursor: 'pointer',
                  fontWeight: '500'
                }}
              >
                🌐 Federation Settings
              </button>
              <button
                onClick={() => setActiveTab('requests')}
                style={{
                  width: '100%',
                  padding: '12px',
                  background: moderationRequests.length > 0 ? 'rgba(255,68,68,0.1)' : 'rgba(255,255,255,0.05)',
                  border: `1px solid ${moderationRequests.length > 0 ? '#ff4444' : '#333'}`,
                  color: moderationRequests.length > 0 ? '#ff4444' : '#888',
                  borderRadius: '8px',
                  cursor: 'pointer',
                  fontWeight: '500'
                }}
              >
                Review Requests {moderationRequests.length > 0 && `(${moderationRequests.length})`}
              </button>
            </div>
          </div>

          <div className="sidebar-card">
            <h3>Admin Info</h3>
            <div style={{ marginTop: '16px' }}>
              <p style={{ color: '#888', fontSize: '13px' }}>Logged in as:</p>
              <p style={{ color: '#ff4444', fontWeight: '600' }}>@{userData?.username}</p>
              <p style={{ color: '#666', fontSize: '12px', marginTop: '8px' }}>
                Full admin privileges enabled
              </p>
            </div>
          </div>
        </aside>
      </div>

      {/* Ban Modal */}
      {showBanModal && banTarget && (
        <div style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          background: 'rgba(0,0,0,0.8)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          zIndex: 2000
        }}>
          <div style={{
            background: '#1a1a2e',
            border: '1px solid #ff4444',
            borderRadius: '16px',
            padding: '24px',
            width: '400px',
            maxWidth: '90%'
          }}>
            <h2 style={{ color: '#ff4444', marginBottom: '16px' }}>Ban User</h2>
            <div style={{ marginBottom: '16px' }}>
              <p style={{ color: '#888', marginBottom: '8px' }}>You are about to suspend:</p>
              <p style={{ color: '#fff', fontWeight: '600', fontSize: '18px' }}>
                @{banTarget.username}
              </p>
            </div>
            <div style={{ marginBottom: '16px' }}>
              <label style={{ color: '#888', display: 'block', marginBottom: '8px' }}>
                Reason for ban (optional):
              </label>
              <textarea
                value={banReason}
                onChange={(e) => setBanReason(e.target.value)}
                placeholder="Violation of community guidelines, spam, harassment, etc."
                style={{
                  width: '100%',
                  padding: '12px',
                  background: 'rgba(0,0,0,0.3)',
                  border: '1px solid #333',
                  borderRadius: '8px',
                  color: '#fff',
                  resize: 'vertical',
                  minHeight: '80px'
                }}
              />
            </div>
            <div style={{ display: 'flex', gap: '12px' }}>
              <button
                onClick={() => {
                  setShowBanModal(false);
                  setBanTarget(null);
                  setBanReason('');
                }}
                style={{
                  flex: 1,
                  padding: '12px',
                  background: 'rgba(255,255,255,0.05)',
                  border: '1px solid #333',
                  color: '#888',
                  borderRadius: '8px',
                  cursor: 'pointer'
                }}
              >
                Cancel
              </button>
              <button
                onClick={handleSuspendUser}
                disabled={actionLoading === banTarget.id}
                style={{
                  flex: 1,
                  padding: '12px',
                  background: 'rgba(255,68,68,0.2)',
                  border: '1px solid #ff4444',
                  color: '#ff4444',
                  borderRadius: '8px',
                  cursor: 'pointer',
                  fontWeight: '600'
                }}
              >
                {actionLoading === banTarget.id ? 'Banning...' : 'Confirm Ban'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
