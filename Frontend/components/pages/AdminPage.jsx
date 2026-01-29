'use client';

import { useState, useEffect } from 'react';
import '../styles/HomePage.css';
import { adminApi } from '@/lib/api';

export default function AdminPage({ onNavigate, userData, isDarkMode, toggleTheme }) {
  const [activeTab, setActiveTab] = useState('users');
  const [users, setUsers] = useState([]);
  const [moderationRequests, setModerationRequests] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [totalUsers, setTotalUsers] = useState(0);
  const [page, setPage] = useState(0);
  const [actionLoading, setActionLoading] = useState(null);

  useEffect(() => {
    if (activeTab === 'users') {
      fetchUsers();
    } else if (activeTab === 'moderation') {
      fetchModerationRequests();
    }
  }, [activeTab, page]);

  const fetchUsers = async () => {
    setIsLoading(true);
    try {
      const result = await adminApi.getAllUsers(50, page * 50);
      setUsers(result.users || []);
      setTotalUsers(result.total || 0);
    } catch (err) {
      console.error('Failed to fetch users:', err);
      alert('Failed to fetch users: ' + err.message);
    } finally {
      setIsLoading(false);
    }
  };

  const fetchModerationRequests = async () => {
    setIsLoading(true);
    try {
      const result = await adminApi.getModerationRequests();
      setModerationRequests(result.requests || []);
    } catch (err) {
      console.error('Failed to fetch moderation requests:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleApproveModeration = async (userId) => {
    setActionLoading(userId);
    try {
      await adminApi.approveModerationRequest(userId);
      alert('Moderation request approved!');
      fetchModerationRequests();
    } catch (err) {
      alert('Failed to approve: ' + err.message);
    } finally {
      setActionLoading(null);
    }
  };

  const handleRejectModeration = async (userId) => {
    setActionLoading(userId);
    try {
      await adminApi.rejectModerationRequest(userId);
      alert('Moderation request rejected.');
      fetchModerationRequests();
    } catch (err) {
      alert('Failed to reject: ' + err.message);
    } finally {
      setActionLoading(null);
    }
  };

  const handleSuspendUser = async (userId) => {
    if (!confirm('Are you sure you want to suspend this user?')) return;
    setActionLoading(userId);
    try {
      await adminApi.suspendUser(userId);
      alert('User suspended.');
      fetchUsers();
    } catch (err) {
      alert('Failed to suspend: ' + err.message);
    } finally {
      setActionLoading(null);
    }
  };

  const handleUnsuspendUser = async (userId) => {
    setActionLoading(userId);
    try {
      await adminApi.unsuspendUser(userId);
      alert('User unsuspended.');
      fetchUsers();
    } catch (err) {
      alert('Failed to unsuspend: ' + err.message);
    } finally {
      setActionLoading(null);
    }
  };

  const handleUpdateRole = async (userId, newRole) => {
    if (!confirm(`Change user role to ${newRole}?`)) return;
    setActionLoading(userId);
    try {
      await adminApi.updateUserRole(userId, newRole);
      alert(`User role updated to ${newRole}.`);
      fetchUsers();
    } catch (err) {
      alert('Failed to update role: ' + err.message);
    } finally {
      setActionLoading(null);
    }
  };

  const formatDate = (isoString) => {
    if (!isoString) return 'N/A';
    return new Date(isoString).toLocaleDateString();
  };

  // Check if current user is admin
  if (userData?.role !== 'admin') {
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
        <p style={{ color: '#666', marginBottom: '24px' }}>You need admin privileges to access this page.</p>
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

  return (
    <div style={{ minHeight: '100vh', background: '#0f0f1a', color: '#fff' }}>
      {/* Header */}
      <nav style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '16px 24px',
        background: 'rgba(0,0,0,0.5)',
        borderBottom: '1px solid #333'
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          <button
            onClick={() => onNavigate('home')}
            style={{
              padding: '8px 16px',
              background: 'rgba(0,217,255,0.1)',
              border: '1px solid #00d9ff',
              color: '#00d9ff',
              borderRadius: '6px',
              cursor: 'pointer'
            }}
          >
            ← Back
          </button>
          <h1 style={{ margin: 0, fontSize: '24px' }}>👑 Admin Dashboard</h1>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <span style={{ color: '#666' }}>Logged in as</span>
          <span style={{ color: '#ff4444', fontWeight: '600' }}>@{userData?.username}</span>
        </div>
      </nav>

      {/* Tabs */}
      <div style={{
        display: 'flex',
        gap: '8px',
        padding: '16px 24px',
        borderBottom: '1px solid #333'
      }}>
        <button
          onClick={() => setActiveTab('users')}
          style={{
            padding: '10px 20px',
            background: activeTab === 'users' ? 'rgba(0,217,255,0.2)' : 'transparent',
            border: `1px solid ${activeTab === 'users' ? '#00d9ff' : '#333'}`,
            color: activeTab === 'users' ? '#00d9ff' : '#666',
            borderRadius: '6px',
            cursor: 'pointer',
            fontWeight: activeTab === 'users' ? '600' : '400'
          }}
        >
          👥 All Users ({totalUsers})
        </button>
        <button
          onClick={() => setActiveTab('moderation')}
          style={{
            padding: '10px 20px',
            background: activeTab === 'moderation' ? 'rgba(0,217,255,0.2)' : 'transparent',
            border: `1px solid ${activeTab === 'moderation' ? '#00d9ff' : '#333'}`,
            color: activeTab === 'moderation' ? '#00d9ff' : '#666',
            borderRadius: '6px',
            cursor: 'pointer',
            fontWeight: activeTab === 'moderation' ? '600' : '400'
          }}
        >
          🛡️ Moderation Requests ({moderationRequests.length})
        </button>
      </div>

      {/* Content */}
      <div style={{ padding: '24px' }}>
        {isLoading ? (
          <div style={{ textAlign: 'center', padding: '40px', color: '#666' }}>
            Loading...
          </div>
        ) : activeTab === 'users' ? (
          <>
            {/* Users Table */}
            <div style={{
              background: 'rgba(255,255,255,0.02)',
              border: '1px solid #333',
              borderRadius: '12px',
              overflow: 'hidden'
            }}>
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr style={{ background: 'rgba(0,0,0,0.3)' }}>
                    <th style={{ padding: '16px', textAlign: 'left', color: '#888', fontWeight: '500' }}>User</th>
                    <th style={{ padding: '16px', textAlign: 'left', color: '#888', fontWeight: '500' }}>Server</th>
                    <th style={{ padding: '16px', textAlign: 'left', color: '#888', fontWeight: '500' }}>Role</th>
                    <th style={{ padding: '16px', textAlign: 'left', color: '#888', fontWeight: '500' }}>Status</th>
                    <th style={{ padding: '16px', textAlign: 'left', color: '#888', fontWeight: '500' }}>Joined</th>
                    <th style={{ padding: '16px', textAlign: 'right', color: '#888', fontWeight: '500' }}>Actions</th>
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
                            background: 'linear-gradient(135deg, #00d9ff, #00ff88)',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center'
                          }}>
                            {user.avatar_url || '👤'}
                          </div>
                          <div>
                            <div style={{ fontWeight: '600' }}>{user.display_name || user.username}</div>
                            <div style={{ color: '#666', fontSize: '12px' }}>@{user.username}</div>
                          </div>
                        </div>
                      </td>
                      <td style={{ padding: '16px', color: '#888' }}>{user.instance_domain}</td>
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
                          <span style={{ color: '#ff4444' }}>🚫 Suspended</span>
                        ) : (
                          <span style={{ color: '#00ff88' }}>✓ Active</span>
                        )}
                      </td>
                      <td style={{ padding: '16px', color: '#888' }}>{formatDate(user.created_at)}</td>
                      <td style={{ padding: '16px', textAlign: 'right' }}>
                        <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
                          {user.role !== 'admin' && (
                            <>
                              {user.is_suspended ? (
                                <button
                                  onClick={() => handleUnsuspendUser(user.id)}
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
                                  Unsuspend
                                </button>
                              ) : (
                                <button
                                  onClick={() => handleSuspendUser(user.id)}
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
                                  Suspend
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
                            </>
                          )}
                        </div>
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
        ) : (
          /* Moderation Requests */
          <div>
            {moderationRequests.length === 0 ? (
              <div style={{
                textAlign: 'center',
                padding: '60px',
                background: 'rgba(255,255,255,0.02)',
                border: '1px solid #333',
                borderRadius: '12px'
              }}>
                <div style={{ fontSize: '48px', marginBottom: '16px' }}>✨</div>
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
                        width: '50px',
                        height: '50px',
                        borderRadius: '50%',
                        background: 'linear-gradient(135deg, #00d9ff, #00ff88)',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontSize: '24px'
                      }}>
                        {user.avatar_url || '👤'}
                      </div>
                      <div>
                        <div style={{ fontWeight: '600', fontSize: '18px' }}>
                          {user.display_name || user.username}
                        </div>
                        <div style={{ color: '#666' }}>@{user.username}@{user.instance_domain}</div>
                        <div style={{ color: '#888', fontSize: '12px', marginTop: '4px' }}>
                          Joined: {formatDate(user.created_at)}
                        </div>
                      </div>
                    </div>
                    <div style={{ display: 'flex', gap: '12px' }}>
                      <button
                        onClick={() => handleApproveModeration(user.id)}
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
                        ✓ Approve
                      </button>
                      <button
                        onClick={() => handleRejectModeration(user.id)}
                        disabled={actionLoading === user.id}
                        style={{
                          padding: '10px 20px',
                          background: 'rgba(255,68,68,0.2)',
                          border: '1px solid #ff4444',
                          color: '#ff4444',
                          borderRadius: '6px',
                          cursor: 'pointer',
                          fontWeight: '600'
                        }}
                      >
                        ✕ Reject
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
